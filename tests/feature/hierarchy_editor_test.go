package feature

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/elcomware/edupilot/modules/tenancy/app/models"
	"github.com/elcomware/edupilot/modules/tenancy/app/services"
	"github.com/elcomware/edupilot/modules/tenancy/providers"
)

// Removing the CAS, tenant predicate, active ancestry check or atomic event
// must break this fixture against the installed schema on either engine.
func verifyHierarchyEditor(t *testing.T, catalog *database.Catalog, tenant scope.TenantID) {
	t.Helper()
	ctx := t.Context()
	editor := requireValue(providers.HierarchyEditor(facades.App()))
	reader := services.HierarchyReader{Pools: catalog}
	directory := services.PersistedDirectory{Pools: catalog}
	exec := func(p database.Pool, q string, args ...any) {
		t.Helper()
		if _, err := p.DB.ExecContext(ctx, p.Bind(q), args...); err != nil {
			t.Fatal(q, err)
		}
	}
	scalar := func(p database.Pool, q string, args ...any) int {
		t.Helper()
		var n int
		if err := p.DB.QueryRowContext(ctx, p.Bind(q), args...).Scan(&n); err != nil {
			t.Fatal(q, err)
		}
		return n
	}
	missing := requireValue(scope.NewID[scope.Tenant]())
	request := services.HierarchyEdit{Tenant: missing, Kind: "organizations", ID: requireValue(scope.NewID[scope.Organization]()).String(), Metadata: services.HierarchyMetadata{Name: "Name"}, Epoch: 3, Version: 1}
	if _, err := editor.Edit(ctx, request); err == nil {
		t.Fatal("accepted unpinned tenant")
	}
	if scalar(catalog.Control, "SELECT count(*) FROM edu_tenant_placement WHERE tenant_id=?", missing.String()) != 0 {
		t.Fatal("created a pin")
	}
	p := requireValue(catalog.ForTenantContext(ctx, tenant.String()))
	// Replace only the preceding fixture's disposable dedicated tenant.
	for _, table := range []string{"tenancy_campuses", "tenancy_schools", "tenancy_organizations", "edu_outbox", "tenant_gates"} {
		exec(p, "DELETE FROM "+table+" WHERE tenant_id=?", tenant.String())
	}
	exec(p, "INSERT INTO tenant_gates (tenant_id,pool_id,placement_epoch,hierarchy_version,state) SELECT ?,pool_id,3,1,'active' FROM edu_pool_marker WHERE singleton=1", tenant.String())
	version := uint64(1)
	orgs := services.OrganizationWriter{Pools: catalog}
	schools := services.SchoolWriter{Pools: catalog}
	campuses := services.CampusWriter{Pools: catalog}
	root := requireValue(models.NewOrganization(requireValue(scope.NewID[scope.Organization]()), tenant, scope.OrganizationID{}, "Root"))
	child := requireValue(models.NewOrganization(requireValue(scope.NewID[scope.Organization]()), tenant, root.ID(), "Child"))
	for _, node := range []models.Organization{root, child} {
		if _, err := orgs.Create(ctx, node, 3, version); err != nil {
			t.Fatal(err)
		}
		version++
	}
	school := requireValue(models.NewSchool(requireValue(scope.NewID[scope.School]()), tenant, child.ID(), "MAIN", "School"))
	sibling := requireValue(models.NewSchool(requireValue(scope.NewID[scope.School]()), tenant, child.ID(), "TAKEN", "Sibling"))
	otherSchool := requireValue(models.NewSchool(requireValue(scope.NewID[scope.School]()), tenant, root.ID(), "OTHER", "Other"))
	for _, node := range []models.School{school, sibling, otherSchool} {
		if _, err := schools.Create(ctx, node, 3, version); err != nil {
			t.Fatal(err)
		}
		version++
	}
	campus := requireValue(models.NewCampus(requireValue(scope.NewID[scope.Campus]()), tenant, school.ID(), "MAIN", "Campus", "UTC"))
	siblingCampus := requireValue(models.NewCampus(requireValue(scope.NewID[scope.Campus]()), tenant, school.ID(), "TAKEN", "Sibling", "UTC"))
	otherCampus := requireValue(models.NewCampus(requireValue(scope.NewID[scope.Campus]()), tenant, otherSchool.ID(), "OTHER", "Other", "UTC"))
	for _, node := range []models.Campus{campus, siblingCampus, otherCampus} {
		if _, err := campuses.Create(ctx, node, 3, version); err != nil {
			t.Fatal(err)
		}
		version++
	}
	assertState := func() {
		t.Helper()
		if scalar(p, "SELECT hierarchy_version FROM tenant_gates WHERE tenant_id=?", tenant.String()) != int(version) || scalar(p, "SELECT count(*) FROM edu_outbox WHERE tenant_id=?", tenant.String()) != int(version)-1 {
			t.Fatal("version/event side effect")
		}
	}
	cases := []struct {
		kind, id, event      string
		before, input, after services.HierarchyMetadata
		selection            scope.Selection
	}{
		{"organizations", child.ID().String(), "organization", services.HierarchyMetadata{Name: "Child"}, services.HierarchyMetadata{Name: "  École group  "}, services.HierarchyMetadata{Name: "École group"}, scope.Selection{Tenant: tenant, Organization: child.ID()}},
		{"schools", school.ID().String(), "school", services.HierarchyMetadata{Name: "School", Code: "MAIN"}, services.HierarchyMetadata{Name: "  École  ", Code: " next "}, services.HierarchyMetadata{Name: "École", Code: "NEXT"}, scope.Selection{Tenant: tenant, School: school.ID()}},
		{"campuses", campus.ID().String(), "campus", services.HierarchyMetadata{Name: "Campus", Code: "MAIN", Timezone: "UTC"}, services.HierarchyMetadata{Name: "  North  ", Code: " north ", Timezone: "Europe/Paris"}, services.HierarchyMetadata{Name: "North", Code: "NORTH", Timezone: "Europe/Paris"}, scope.Selection{Tenant: tenant, Campus: campus.ID()}},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			beforePath := requireValue(directory.Resolve(ctx, tc.selection))
			q := services.HierarchyQuery{Tenant: tenant, Kind: tc.kind, Limit: 1}
			page := requireValue(reader.List(ctx, q))
			if page.NextCursor == "" {
				t.Fatal("missing cursor fixture")
			}
			cmd := services.HierarchyEdit{Tenant: tenant, Kind: tc.kind, ID: tc.id, Metadata: tc.input, Epoch: 3, Version: version}
			// Every kind validates even a no-op behind the same placement fence.
			for _, bounds := range [][2]uint64{{0, version}, {math.MaxUint64, version}, {3, 0}, {3, math.MaxInt64}, {3, math.MaxUint64}} {
				bad := cmd
				bad.Epoch, bad.Version = bounds[0], bounds[1]
				if _, err := editor.Edit(ctx, bad); !errors.Is(err, models.ErrInvalid) {
					t.Fatal("invalid bounds", err)
				}
			}
			for _, bounds := range [][2]uint64{{2, version}, {3, version - 1}} {
				bad := cmd
				bad.Epoch, bad.Version = bounds[0], bounds[1]
				if _, err := editor.Edit(ctx, bad); !errors.Is(err, models.ErrConflict) {
					t.Fatal("stale", err)
				}
			}
			for _, state := range []string{"maintenance", "suspended", "retired", "provisioning"} {
				exec(p, "UPDATE tenant_gates SET state=? WHERE tenant_id=?", state, tenant.String())
				if _, err := editor.Edit(ctx, cmd); !errors.Is(err, models.ErrConflict) {
					t.Fatal(state, err)
				}
			}
			exec(p, "UPDATE tenant_gates SET state='active',pool_id='wrong' WHERE tenant_id=?", tenant.String())
			if _, err := editor.Edit(ctx, cmd); !errors.Is(err, models.ErrConflict) {
				t.Fatal("wrong gate marker", err)
			}
			exec(p, "UPDATE tenant_gates SET pool_id=(SELECT pool_id FROM edu_pool_marker WHERE singleton=1) WHERE tenant_id=?", tenant.String())
			var poolID, poolOwner string
			if err := p.DB.QueryRow("SELECT pool_id,owner FROM edu_pool_marker WHERE singleton=1").Scan(&poolID, &poolOwner); err != nil {
				t.Fatal(err)
			}
			exec(catalog.Control, "UPDATE edu_tenant_placement SET pool_id='wrong' WHERE tenant_id=?", tenant.String())
			if _, err := editor.Edit(ctx, cmd); !errors.Is(err, database.ErrPlacement) {
				t.Fatal("wrong pin", err)
			}
			exec(catalog.Control, "UPDATE edu_tenant_placement SET pool_id=? WHERE tenant_id=?", poolID, tenant.String())
			exec(p, "UPDATE edu_pool_marker SET pool_id='wrong'")
			if _, err := editor.Edit(ctx, cmd); err == nil {
				t.Fatal("wrong physical marker")
			}
			exec(p, "UPDATE edu_pool_marker SET pool_id=?", poolID)
			exec(p, "DELETE FROM edu_pool_marker WHERE singleton=1")
			if _, err := editor.Edit(ctx, cmd); err == nil {
				t.Fatal("missing physical marker")
			}
			exec(p, "INSERT INTO edu_pool_marker (singleton,owner,pool_id) VALUES (1,?,?)", poolOwner, poolID)
			exec(p, "UPDATE edu_pool_marker SET owner='tenant:wrong'")
			if _, err := editor.Edit(ctx, cmd); !errors.Is(err, database.ErrPlacement) {
				t.Fatal("wrong physical owner", err)
			}
			exec(p, "UPDATE edu_pool_marker SET owner=?", poolOwner)
			// All organizations in the path, the school and the selected campus participate.
			ancestors := []struct{ table, id string }{{"tenancy_organizations", root.ID().String()}, {"tenancy_organizations", child.ID().String()}}
			if tc.kind != "organizations" {
				ancestors = append(ancestors, struct{ table, id string }{"tenancy_schools", school.ID().String()})
			}
			if tc.kind == "campuses" {
				ancestors = append(ancestors, struct{ table, id string }{"tenancy_campuses", campus.ID().String()})
			}
			for _, a := range ancestors {
				exec(p, "UPDATE "+a.table+" SET active=false WHERE tenant_id=? AND id=?", tenant.String(), a.id)
				if _, err := editor.Edit(ctx, cmd); !errors.Is(err, models.ErrScope) {
					t.Fatal("inactive path", a, err)
				}
				exec(p, "UPDATE "+a.table+" SET active=true WHERE tenant_id=? AND id=?", tenant.String(), a.id)
			}
			invalid := []services.HierarchyMetadata{{}, {Name: "bad\nname", Code: tc.input.Code, Timezone: tc.input.Timezone}, {Name: "Name", Code: "bad code", Timezone: "Local"}}
			if tc.kind == "schools" {
				invalid = append(invalid, services.HierarchyMetadata{Name: "Name", Code: "OK", Timezone: "UTC"})
			}
			if tc.kind == "campuses" {
				invalid = append(invalid, services.HierarchyMetadata{Name: "Name", Code: "OK", Timezone: "Invalid/Zone"}, services.HierarchyMetadata{Name: "Name", Code: "OK", Timezone: "Local"})
			}
			for _, metadata := range invalid {
				bad := cmd
				bad.Metadata = metadata
				if _, err := editor.Edit(ctx, bad); !errors.Is(err, models.ErrInvalid) {
					t.Fatal("invalid metadata", metadata, err)
				}
			}
			bad := cmd
			bad.ID = requireValue(scope.NewID[scope.Organization]()).String()
			if _, err := editor.Edit(ctx, bad); !errors.Is(err, models.ErrScope) {
				t.Fatal("missing ID", err)
			}
			bad = cmd
			bad.ID = "malformed"
			if _, err := editor.Edit(ctx, bad); !errors.Is(err, models.ErrInvalid) {
				t.Fatal("malformed ID", err)
			}
			// Sibling uniqueness also rejects updates, and rolls back the version/event.
			if tc.kind != "organizations" {
				bad = cmd
				bad.Metadata.Code = " taken "
				if _, err := editor.Edit(ctx, bad); err == nil {
					t.Fatal("duplicate sibling code")
				}
			}
			assertState()
			// Failure after metadata/version SQL must roll back, allowing exact retry.
			exec(p, "ALTER TABLE edu_outbox RENAME COLUMN payload TO unavailable_payload")
			if _, err := editor.Edit(ctx, cmd); err == nil {
				t.Fatal("edit without event")
			}
			exec(p, "ALTER TABLE edu_outbox RENAME COLUMN unavailable_payload TO payload")
			assertState()
			unchanged := cmd
			unchanged.Metadata = tc.before
			receipt, err := editor.Edit(ctx, unchanged)
			if err != nil || receipt.Changed || receipt.Entry.Name != tc.before.Name || receipt.Entry.Code != tc.before.Code || receipt.Entry.Timezone != tc.before.Timezone || !reflect.DeepEqual(receipt.Entry.Path, beforePath) {
				t.Fatal("rollback/no-op", receipt, err)
			}
			assertState()
			receipt, err = editor.Edit(ctx, cmd)
			if err != nil {
				t.Fatal(err)
			}
			version++
			beforePath.HierarchyVersion = version
			if !receipt.Changed || receipt.Entry.ID != tc.id || !receipt.Entry.Active || receipt.Entry.Name != tc.after.Name || receipt.Entry.Code != tc.after.Code || receipt.Entry.Timezone != tc.after.Timezone || !reflect.DeepEqual(receipt.Entry.Path, beforePath) {
				t.Fatal("receipt", receipt, beforePath)
			}
			persisted := requireValue(reader.List(ctx, services.HierarchyQuery{Tenant: tenant, Kind: tc.kind, Limit: 100}))
			found := false
			for _, entry := range persisted.Entries {
				if entry.ID == tc.id {
					found = true
					if !reflect.DeepEqual(entry, receipt.Entry) {
						t.Fatal("persisted metadata", entry, receipt)
					}
				}
			}
			if !found {
				t.Fatal("missing persisted metadata")
			}
			resolved := requireValue(directory.Resolve(ctx, tc.selection))
			if !reflect.DeepEqual(resolved, receipt.Entry.Path) {
				t.Fatal("directory compatibility", resolved)
			}
			var payload []byte
			if err := p.DB.QueryRowContext(ctx, p.Bind("SELECT payload FROM edu_outbox WHERE tenant_id=? AND event_type=? AND aggregate_id=?"), tenant.String(), "tenancy."+tc.event+".updated.v1", tc.id).Scan(&payload); err != nil {
				t.Fatal(err)
			}
			var event services.HierarchyMetadataUpdated
			if err := json.Unmarshal(payload, &event); err != nil || event.Before != tc.before || event.After != tc.after || !reflect.DeepEqual(event.Path, receipt.Entry.Path) {
				t.Fatal("event", string(payload), err)
			}
			if _, err := editor.Edit(ctx, cmd); !errors.Is(err, models.ErrConflict) {
				t.Fatal("stale replay", err)
			}
			cmd.Version = version
			receipt, err = editor.Edit(ctx, cmd)
			if err != nil || receipt.Changed || receipt.Entry.Path.HierarchyVersion != version {
				t.Fatal("normalized no-op", receipt, err)
			}
			assertState()
			q.Cursor = page.NextCursor
			if _, err := reader.List(ctx, q); !errors.Is(err, models.ErrConflict) {
				t.Fatal("old cursor", err)
			}
		})
	}
	// Foreign IDs exist in this same physical pool but must never be selected.
	other := requireValue(scope.NewID[scope.Tenant]())
	foreignOrg := requireValue(scope.NewID[scope.Organization]()).String()
	foreignSchool := requireValue(scope.NewID[scope.School]()).String()
	foreignCampus := requireValue(scope.NewID[scope.Campus]()).String()
	exec(p, "INSERT INTO tenant_gates (tenant_id,pool_id,placement_epoch,hierarchy_version,state) SELECT ?,pool_id,1,4,'active' FROM edu_pool_marker WHERE singleton=1", other.String())
	exec(p, "INSERT INTO tenancy_organizations (tenant_id,id,name,active) VALUES (?,?,'Foreign',true)", other.String(), foreignOrg)
	exec(p, "INSERT INTO tenancy_schools (tenant_id,id,organization_id,code,name,active) VALUES (?,?,?,'FOREIGN','Foreign',true)", other.String(), foreignSchool, foreignOrg)
	exec(p, "INSERT INTO tenancy_campuses (tenant_id,id,school_id,code,name,timezone,active) VALUES (?,?,?,'FOREIGN','Foreign','UTC',true)", other.String(), foreignCampus, foreignSchool)
	for i, tc := range cases {
		cmd := services.HierarchyEdit{Tenant: tenant, Kind: tc.kind, ID: []string{foreignOrg, foreignSchool, foreignCampus}[i], Metadata: tc.input, Epoch: 3, Version: version}
		if _, err := editor.Edit(ctx, cmd); !errors.Is(err, models.ErrScope) {
			t.Fatal("foreign ID", tc.kind, err)
		}
	}
	for _, table := range []string{"tenancy_campuses", "tenancy_schools", "tenancy_organizations", "tenant_gates"} {
		exec(p, "DELETE FROM "+table+" WHERE tenant_id=?", other.String())
	}
	// Codes already owned by a different parent remain valid on update.
	for _, cmd := range []services.HierarchyEdit{
		{Tenant: tenant, Kind: "schools", ID: otherSchool.ID().String(), Metadata: services.HierarchyMetadata{Name: "Other", Code: "NEXT"}, Epoch: 3},
		{Tenant: tenant, Kind: "campuses", ID: otherCampus.ID().String(), Metadata: services.HierarchyMetadata{Name: "Other", Code: "NORTH", Timezone: "UTC"}, Epoch: 3},
	} {
		cmd.Version = version
		if _, err := editor.Edit(ctx, cmd); err != nil {
			t.Fatal("same code other owner", err)
		}
		version++
	}
	for _, kind := range []string{"", "locations"} {
		cmd := request
		cmd.Tenant = tenant
		cmd.Kind = kind
		cmd.Version = version
		if _, err := editor.Edit(ctx, cmd); !errors.Is(err, models.ErrInvalid) {
			t.Fatal("invalid kind", err)
		}
	}

	rootEdit := services.HierarchyEdit{Tenant: tenant, Kind: "organizations", ID: root.ID().String(), Metadata: services.HierarchyMetadata{Name: "Renamed root"}, Epoch: 3, Version: version}
	receipt, err := editor.Edit(ctx, rootEdit)
	if err != nil || !receipt.Changed || receipt.Entry.Path.Organization != root.ID() || !reflect.DeepEqual(receipt.Entry.Path.Ancestors, []scope.OrganizationID{root.ID()}) {
		t.Fatal("root metadata", receipt, err)
	}
	version++
	var parent any
	if err := p.DB.QueryRowContext(ctx, p.Bind("SELECT parent_id FROM tenancy_organizations WHERE tenant_id=? AND id=?"), tenant.String(), root.ID().String()).Scan(&parent); err != nil || parent != nil {
		t.Fatal("root parent", parent, err)
	}
	invalidTenant := rootEdit
	invalidTenant.Tenant = scope.TenantID{}
	if _, err := editor.Edit(ctx, invalidTenant); !errors.Is(err, models.ErrInvalid) {
		t.Fatal("zero tenant", err)
	}
	// Two edits, and then an edit/create, compete for exactly one tenant version.
	for round := 0; round < 2; round++ {
		done := make(chan error, 2)
		expected := version
		for i := 0; i < 2; i++ {
			go func() {
				var err error
				if round == 1 && i == 1 {
					node := requireValue(models.NewOrganization(requireValue(scope.NewID[scope.Organization]()), tenant, root.ID(), "Concurrent"))
					_, err = orgs.Create(ctx, node, 3, expected)
				} else {
					_, err = editor.Edit(ctx, services.HierarchyEdit{Tenant: tenant, Kind: "organizations", ID: child.ID().String(), Metadata: services.HierarchyMetadata{Name: fmt.Sprintf("Race %d %d", round, i)}, Epoch: 3, Version: expected})
				}
				done <- err
			}()
		}
		first, second := <-done, <-done
		if !((first == nil && errors.Is(second, models.ErrConflict)) || (second == nil && errors.Is(first, models.ErrConflict))) {
			t.Fatal("concurrent outcomes", first, second)
		}
		version++
	}
	assertState()
	if p.DB != catalog.Control.DB {
		for _, table := range []string{"tenancy_organizations", "tenancy_schools", "tenancy_campuses", "tenant_gates", "edu_outbox"} {
			if scalar(catalog.Control, "SELECT count(*) FROM "+table+" WHERE tenant_id=?", tenant.String()) != 0 {
				t.Fatal("dedicated data leaked", table)
			}
		}
	}
}
