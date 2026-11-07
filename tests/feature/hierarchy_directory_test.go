package feature

import (
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/elcomware/edupilot/modules/tenancy/app/models"
	"github.com/elcomware/edupilot/modules/tenancy/app/services"
	"github.com/elcomware/edupilot/modules/tenancy/providers"
)

// Removing the keyset bound, owner predicate, gate check or snapshot version
// check must break these tests against the real installed schema on both engines.
func verifyHierarchyReader(t *testing.T, catalog *database.Catalog, tenant scope.TenantID) {
	t.Helper()
	ctx := t.Context()
	reader := requireValue(providers.HierarchyReader(facades.App()))
	exec := func(p database.Pool, query string, args ...any) {
		t.Helper()
		if _, err := p.DB.ExecContext(ctx, p.Bind(query), args...); err != nil {
			t.Fatal(query, err)
		}
	}
	missing := requireValue(scope.NewID[scope.Tenant]())
	if _, err := reader.List(ctx, services.HierarchyQuery{Tenant: missing, Kind: "organizations", Limit: 1}); err == nil {
		t.Fatal("accepted unpinned tenant")
	}
	var pins int
	if err := catalog.Control.DB.QueryRowContext(ctx, catalog.Control.Bind("SELECT count(*) FROM edu_tenant_placement WHERE tenant_id=?"), missing.String()).Scan(&pins); err != nil || pins != 0 {
		t.Fatal("implicit pin", pins, err)
	}
	p := requireValue(catalog.ForTenantContext(ctx, tenant.String()))
	// This dedicated fixture follows the writer fixture and replaces only its synthetic tenant.
	for _, table := range []string{"tenancy_campuses", "tenancy_schools", "tenancy_organizations", "edu_outbox", "tenant_gates"} {
		exec(p, "DELETE FROM "+table+" WHERE tenant_id=?", tenant.String())
	}
	exec(p, "INSERT INTO tenant_gates (tenant_id,pool_id,placement_epoch,hierarchy_version,state) SELECT ?,pool_id,1,1,'active' FROM edu_pool_marker WHERE singleton=1", tenant.String())
	query := services.HierarchyQuery{Tenant: tenant, Kind: "organizations", Limit: 2}
	empty, err := reader.List(ctx, query)
	if err != nil || len(empty.Entries) != 0 || empty.NextCursor != "" || empty.HierarchyVersion != 1 {
		t.Fatal("empty", empty, err)
	}
	orgID := func(n int) scope.OrganizationID {
		return requireValue(scope.Parse[scope.Organization](fmt.Sprintf("00000000-0000-4000-8000-%012d", n)))
	}
	schoolID := func(n int) scope.SchoolID {
		return requireValue(scope.Parse[scope.School](fmt.Sprintf("10000000-0000-4000-8000-%012d", n)))
	}
	campusID := func(n int) scope.CampusID {
		return requireValue(scope.Parse[scope.Campus](fmt.Sprintf("20000000-0000-4000-8000-%012d", n)))
	}
	exec(p, "UPDATE tenant_gates SET hierarchy_version=110 WHERE tenant_id=?", tenant.String())
	for _, n := range []int{1, 3, 2} {
		var parent any
		if n != 1 {
			parent = orgID(1).String()
		}
		exec(p, "INSERT INTO tenancy_organizations (tenant_id,id,parent_id,name,active) VALUES (?,?,?,'Same name',true)", tenant.String(), orgID(n).String(), parent)
	}
	for n := 1; n <= 2; n++ {
		exec(p, "INSERT INTO tenancy_schools (tenant_id,id,organization_id,code,name,active) VALUES (?,?,?,'MAIN','Same name',true)", tenant.String(), schoolID(n).String(), orgID(n+1).String())
	}
	// 101 records exercise the maximum bound as well as a final partial page.
	for n := 101; n >= 1; n-- {
		owner := schoolID(1)
		if n == 101 {
			owner = schoolID(2)
		}
		exec(p, "INSERT INTO tenancy_campuses (tenant_id,id,school_id,code,name,timezone,active) VALUES (?,?,?,?, 'Same name','Africa/Nairobi',true)", tenant.String(), campusID(n).String(), owner.String(), fmt.Sprintf("C%03d", n))
	}
	first, err := reader.List(ctx, query)
	if err != nil || len(first.Entries) != 2 || first.Entries[0].ID != orgID(1).String() || first.Entries[1].ID != orgID(2).String() || first.NextCursor == "" {
		t.Fatal("first page", first, err)
	}
	query.Cursor = first.NextCursor
	second, err := reader.List(ctx, query)
	if err != nil || len(second.Entries) != 1 || second.Entries[0].ID != orgID(3).String() || second.NextCursor != "" {
		t.Fatal("second page", second, err)
	}
	query.Cursor = ""
	filtered := services.HierarchyQuery{Tenant: tenant, Kind: "organizations", Organization: orgID(1), Limit: 100}
	page, err := reader.List(ctx, filtered)
	if err != nil || len(page.Entries) != 2 || page.Entries[0].ID != orgID(2).String() {
		t.Fatal("direct children", page, err)
	}
	schoolQuery := services.HierarchyQuery{Tenant: tenant, Kind: "schools", Organization: orgID(2), Limit: 100}
	page, err = reader.List(ctx, schoolQuery)
	if err != nil || len(page.Entries) != 1 || page.Entries[0].ID != schoolID(1).String() || page.Entries[0].Code != "MAIN" {
		t.Fatal("school owner", page, err)
	}
	campusQuery := services.HierarchyQuery{Tenant: tenant, Kind: "campuses", School: schoolID(1), Limit: 100}
	page, err = reader.List(ctx, campusQuery)
	if err != nil || len(page.Entries) != 100 || page.NextCursor != "" {
		t.Fatal("bounded campuses", len(page.Entries), err)
	}
	for i, entry := range page.Entries {
		if entry.ID != campusID(i+1).String() || entry.Timezone != "Africa/Nairobi" || entry.Name != "Same name" || !entry.Active || !entry.Path.Active || entry.Path.School != schoolID(1) || !reflect.DeepEqual(entry.Path.Ancestors, []scope.OrganizationID{orgID(1), orgID(2)}) {
			t.Fatal("campus metadata", entry)
		}
	}
	campusQuery.School = scope.SchoolID{}
	page, err = reader.List(ctx, campusQuery)
	if err != nil || len(page.Entries) != 100 || page.NextCursor == "" {
		t.Fatal("maximum page", len(page.Entries), err)
	}
	campusQuery.Cursor = page.NextCursor
	page, err = reader.List(ctx, campusQuery)
	if err != nil || len(page.Entries) != 1 || page.Entries[0].ID != campusID(101).String() {
		t.Fatal("last campus", page, err)
	}
	for _, limit := range []int{-1, 0, 101} {
		bad := query
		bad.Limit = limit
		if _, err := reader.List(ctx, bad); !errors.Is(err, models.ErrInvalid) {
			t.Fatal("bad limit", limit, err)
		}
	}
	for _, bad := range []services.HierarchyQuery{
		{Tenant: tenant, Kind: "unknown", Limit: 1}, {Tenant: tenant, Kind: "schools", School: schoolID(1), Limit: 1}, {Tenant: tenant, Kind: "campuses", Organization: orgID(1), Limit: 1},
		{Tenant: tenant, Kind: "organizations", Cursor: "bad", Limit: 1}, {Tenant: tenant, Kind: "organizations", Cursor: strings.Repeat("A", 4097), Limit: 1},
		{Tenant: tenant, Kind: "schools", Cursor: first.NextCursor, Limit: 1}, {Tenant: tenant, Kind: "organizations", Organization: orgID(1), Cursor: first.NextCursor, Limit: 1},
	} {
		if _, err := reader.List(ctx, bad); !errors.Is(err, models.ErrInvalid) {
			t.Fatal("invalid query/cursor", bad, err)
		}
	}
	raw := requireValue(base64.RawURLEncoding.DecodeString(first.NextCursor))
	duplicate := strings.Replace(string(raw), `"format":1`, `"format":1,"format":1`, 1)
	bad := query
	bad.Cursor = base64.RawURLEncoding.EncodeToString([]byte(duplicate))
	if _, err := reader.List(ctx, bad); !errors.Is(err, models.ErrInvalid) {
		t.Error("duplicate cursor fields", err)
	}
	bad.Cursor = base64.RawURLEncoding.EncodeToString(append(raw, []byte("{}")...))
	if _, err := reader.List(ctx, bad); !errors.Is(err, models.ErrInvalid) {
		t.Fatal("trailing cursor", err)
	}
	other := requireValue(scope.NewID[scope.Tenant]())
	requireValue(catalog.ForTenantContext(ctx, other.String()))
	exec(catalog.Control, "INSERT INTO tenant_gates (tenant_id,pool_id,placement_epoch,hierarchy_version,state) SELECT ?,pool_id,1,110,'active' FROM edu_pool_marker WHERE singleton=1", other.String())
	foreign := query
	foreign.Tenant = other
	foreign.Cursor = first.NextCursor
	if _, err := reader.List(ctx, foreign); !errors.Is(err, models.ErrInvalid) {
		t.Fatal("foreign cursor", err)
	}
	foreign.Cursor = ""
	foreign.Organization = orgID(1)
	if _, err := reader.List(ctx, foreign); !errors.Is(err, models.ErrScope) {
		t.Fatal("cross tenant filter", err)
	}
	bad = query
	bad.Organization = orgID(9)
	if _, err := reader.List(ctx, bad); !errors.Is(err, models.ErrScope) {
		t.Fatal("missing filter", err)
	}
	// Same-ID rows in shared storage cannot leak between tenants.
	exec(catalog.Control, "INSERT INTO tenancy_organizations (tenant_id,id,name,active) VALUES (?,?,'Other tenant',true)", other.String(), orgID(1).String())
	foreign.Organization = scope.OrganizationID{}
	page, err = reader.List(ctx, foreign)
	if err != nil || len(page.Entries) != 1 || page.Entries[0].Name != "Other tenant" {
		t.Fatal("tenant isolation", page, err)
	}
	for _, table := range []string{"tenancy_organizations", "tenancy_schools", "tenancy_campuses"} {
		exec(p, "UPDATE "+table+" SET active=false WHERE tenant_id=?", tenant.String())
		page, err = reader.List(ctx, services.HierarchyQuery{Tenant: tenant, Kind: "campuses", Limit: 1})
		if err != nil || len(page.Entries) != 1 || page.Entries[0].Path.Active {
			t.Fatal("effective inactive ancestry", table, page, err)
		}
		if table != "tenancy_campuses" && !page.Entries[0].Active {
			t.Fatal("own active flag lost", table)
		}
		exec(p, "UPDATE "+table+" SET active=true WHERE tenant_id=?", tenant.String())
	}
	for _, state := range []string{"maintenance", "suspended", "retired", "provisioning"} {
		exec(p, "UPDATE tenant_gates SET state=? WHERE tenant_id=?", state, tenant.String())
		if _, err := reader.List(ctx, query); !errors.Is(err, models.ErrScope) {
			t.Fatal("closed gate", state, err)
		}
	}
	exec(p, "UPDATE tenant_gates SET state='active',pool_id='wrong' WHERE tenant_id=?", tenant.String())
	if _, err := reader.List(ctx, query); err == nil {
		t.Fatal("wrong physical gate")
	}
	exec(p, "UPDATE tenant_gates SET pool_id=(SELECT pool_id FROM edu_pool_marker WHERE singleton=1) WHERE tenant_id=?", tenant.String())
	exec(catalog.Control, "UPDATE edu_tenant_placement SET pool_id='wrong' WHERE tenant_id=?", tenant.String())
	if _, err := reader.List(ctx, query); !errors.Is(err, database.ErrPlacement) {
		t.Fatal("wrong placement", err)
	}
	var poolID string
	if err := p.DB.QueryRow("SELECT pool_id FROM edu_pool_marker WHERE singleton=1").Scan(&poolID); err != nil {
		t.Fatal(err)
	}
	exec(catalog.Control, "UPDATE edu_tenant_placement SET pool_id=? WHERE tenant_id=?", poolID, tenant.String())
	exec(p, "UPDATE tenant_gates SET placement_epoch=2 WHERE tenant_id=?", tenant.String())
	bad = query
	bad.Cursor = first.NextCursor
	if _, err := reader.List(ctx, bad); !errors.Is(err, models.ErrConflict) {
		t.Fatal("stale placement epoch", err)
	}
	exec(p, "UPDATE tenant_gates SET placement_epoch=1 WHERE tenant_id=?", tenant.String())
	// The physical identity also binds a cursor when the hierarchy version and epoch coincide.
	movedPool := requireValue(scope.NewID[scope.Pool]()).String()
	exec(p, "UPDATE edu_pool_marker SET pool_id=? WHERE singleton=1", movedPool)
	exec(p, "UPDATE tenant_gates SET pool_id=? WHERE tenant_id=?", movedPool, tenant.String())
	exec(catalog.Control, "UPDATE edu_tenant_placement SET pool_id=? WHERE tenant_id=?", movedPool, tenant.String())
	if _, err := reader.List(ctx, bad); !errors.Is(err, models.ErrConflict) {
		t.Fatal("stale physical identity", err)
	}
	exec(p, "UPDATE edu_pool_marker SET pool_id=? WHERE singleton=1", poolID)
	exec(p, "UPDATE tenant_gates SET pool_id=? WHERE tenant_id=?", poolID, tenant.String())
	exec(catalog.Control, "UPDATE edu_tenant_placement SET pool_id=? WHERE tenant_id=?", poolID, tenant.String())
	var owner string
	if err := p.DB.QueryRow("SELECT owner FROM edu_pool_marker WHERE singleton=1").Scan(&owner); err != nil {
		t.Fatal(err)
	}
	exec(p, "UPDATE edu_pool_marker SET owner='tenant:foreign' WHERE singleton=1")
	if _, err := reader.List(ctx, query); !errors.Is(err, database.ErrPlacement) {
		t.Fatal("foreign pool owner", err)
	}
	exec(p, "UPDATE edu_pool_marker SET owner=? WHERE singleton=1", owner)
	// A writer committed between pages invalidates the old continuation.
	org := requireValue(models.NewOrganization(orgID(4), tenant, scope.OrganizationID{}, "Concurrent"))
	done := make(chan error, 1)
	go func() {
		_, err := (services.OrganizationWriter{Pools: catalog}).Create(ctx, org, 1, 110)
		done <- err
	}()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	bad = query
	bad.Cursor = first.NextCursor
	if _, err := reader.List(ctx, bad); !errors.Is(err, models.ErrConflict) {
		t.Fatal("stale continuation", err)
	}
	bad = query
	bad.Version = 110
	if _, err := reader.List(ctx, bad); !errors.Is(err, models.ErrConflict) {
		t.Fatal("stale requested version", err)
	}
	// Listing has emitted no events and performed no hierarchy edits of its own.
	var events, version int
	if err := p.DB.QueryRowContext(ctx, p.Bind("SELECT count(*) FROM edu_outbox WHERE tenant_id=?"), tenant.String()).Scan(&events); err != nil || events != 1 {
		t.Fatal("read side effects", events, err)
	}
	if err := p.DB.QueryRowContext(ctx, p.Bind("SELECT hierarchy_version FROM tenant_gates WHERE tenant_id=?"), tenant.String()).Scan(&version); err != nil || version != 111 {
		t.Fatal("read version mutation", version, err)
	}
	exec(p, "DELETE FROM tenancy_campuses WHERE tenant_id=?", tenant.String())
	exec(p, "DELETE FROM tenancy_schools WHERE tenant_id=?", tenant.String())
	exec(p, "DELETE FROM tenancy_organizations WHERE tenant_id=?", tenant.String())
	exec(p, "DELETE FROM tenant_gates WHERE tenant_id=?", tenant.String())
	// A legacy Auth school realm with a placement is insufficient to list hierarchy.
	exec(catalog.Control, "INSERT INTO auth_realms (school_id) VALUES (?)", tenant.String())
	if _, err := reader.List(ctx, query); !errors.Is(err, models.ErrScope) {
		t.Fatal("legacy realm", err)
	}
}
