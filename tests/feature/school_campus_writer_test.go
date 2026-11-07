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

const hierarchyTenantID = "5f0a174e-999f-4c61-aeee-0a6b992fa204"

func verifySchoolCampusWriters(t *testing.T, catalog *database.Catalog, tenant scope.TenantID) {
	t.Helper()
	ctx := t.Context()
	schools := requireValue(providers.SchoolWriter(facades.App()))
	campuses := requireValue(providers.CampusWriter(facades.App()))
	directory := services.PersistedDirectory{Pools: catalog}
	root := requireValue(models.NewOrganization(requireValue(scope.NewID[scope.Organization]()), tenant, scope.OrganizationID{}, "Group"))
	child := requireValue(models.NewOrganization(requireValue(scope.NewID[scope.Organization]()), tenant, root.ID(), "Division"))
	school := func(parent scope.OrganizationID, code string) models.School {
		return requireValue(models.NewSchool(requireValue(scope.NewID[scope.School]()), tenant, parent, code, " School "))
	}
	campus := func(parent scope.SchoolID, code string) models.Campus {
		return requireValue(models.NewCampus(requireValue(scope.NewID[scope.Campus]()), tenant, parent, code, " Campus ", "Africa/Nairobi"))
	}
	firstSchool := school(child.ID(), " main ")
	firstCampus := campus(firstSchool.ID(), " main ")
	for name, create := range map[string]func() (scope.Chain, error){
		"school": func() (scope.Chain, error) { return schools.Create(ctx, firstSchool, 3, 1) },
		"campus": func() (scope.Chain, error) { return campuses.Create(ctx, firstCampus, 3, 1) },
	} {
		if _, err := create(); err == nil {
			t.Fatal(name, "created an unpinned tenant")
		}
	}
	var pins int
	if err := catalog.Control.DB.QueryRowContext(ctx, catalog.Control.Bind("SELECT count(*) FROM edu_tenant_placement WHERE tenant_id=?"), tenant.String()).Scan(&pins); err != nil || pins != 0 {
		t.Fatal("writer registered missing placement", pins, err)
	}
	p := requireValue(catalog.ForTenantContext(ctx, tenant.String()))
	exec := func(pool database.Pool, q string, args ...any) {
		t.Helper()
		if _, err := pool.DB.ExecContext(ctx, pool.Bind(q), args...); err != nil {
			t.Fatal(err)
		}
	}
	scalar := func(pool database.Pool, q string, args ...any) int {
		t.Helper()
		var n int
		if err := pool.DB.QueryRowContext(ctx, pool.Bind(q), args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	exec(p, "INSERT INTO tenant_gates (tenant_id,pool_id,placement_epoch,hierarchy_version,state) SELECT ?,pool_id,3,1,'active' FROM edu_pool_marker WHERE singleton=1", tenant.String())
	orgs := services.OrganizationWriter{Pools: catalog}
	if _, err := orgs.Create(ctx, root, 3, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := orgs.Create(ctx, child, 3, 2); err != nil {
		t.Fatal(err)
	}
	version := uint64(3)
	assertState := func(schoolCount, campusCount int) {
		t.Helper()
		for query, want := range map[string]int{
			"SELECT hierarchy_version FROM tenant_gates WHERE tenant_id=?": int(version),
			"SELECT count(*) FROM tenancy_schools WHERE tenant_id=?":       schoolCount,
			"SELECT count(*) FROM tenancy_campuses WHERE tenant_id=?":      campusCount,
			"SELECT count(*) FROM edu_outbox WHERE tenant_id=?":            int(version) - 1,
		} {
			if got := scalar(p, query, tenant.String()); got != want {
				t.Fatalf("%s: got %d want %d", query, got, want)
			}
		}
	}
	checkChain := func(chain scope.Chain, kind, id string) {
		t.Helper()
		if chain.Tenant != tenant || !chain.Active || chain.HierarchyVersion != version || !reflect.DeepEqual(chain.Ancestors, []scope.OrganizationID{root.ID(), child.ID()}) {
			t.Fatal("incorrect ancestry", chain)
		}
		resolved, err := directory.Resolve(ctx, scope.Selection{Tenant: tenant, School: chain.School, Campus: chain.Campus})
		if err != nil || !reflect.DeepEqual(resolved, chain) {
			t.Fatal("lookup", resolved, chain, err)
		}
		var payload []byte
		if err := p.DB.QueryRowContext(ctx, p.Bind("SELECT payload FROM edu_outbox WHERE tenant_id=? AND event_type=? AND aggregate_id=?"), tenant.String(), "tenancy."+kind+".created.v1", id).Scan(&payload); err != nil {
			t.Fatal(err)
		}
		var event scope.Chain
		if err := json.Unmarshal(payload, &event); err != nil || !reflect.DeepEqual(event, chain) {
			t.Fatal("event chain", string(payload), chain, err)
		}
	}
	for _, bounds := range [][2]uint64{{0, version}, {math.MaxUint64, version}, {3, 0}, {3, math.MaxInt64}, {3, math.MaxUint64}} {
		if _, err := schools.Create(ctx, firstSchool, bounds[0], bounds[1]); !errors.Is(err, models.ErrInvalid) {
			t.Fatal("school invalid bounds", bounds, err)
		}
		if _, err := campuses.Create(ctx, firstCampus, bounds[0], bounds[1]); !errors.Is(err, models.ErrInvalid) {
			t.Fatal("campus invalid bounds", bounds, err)
		}
	}
	for _, invalid := range []models.School{{}, firstSchool.WithActive(false)} {
		if _, err := schools.Create(ctx, invalid, 3, version); !errors.Is(err, models.ErrInvalid) {
			t.Fatal("invalid school", err)
		}
	}
	for _, invalid := range []models.Campus{{}, firstCampus.WithActive(false)} {
		if _, err := campuses.Create(ctx, invalid, 3, version); !errors.Is(err, models.ErrInvalid) {
			t.Fatal("invalid campus", err)
		}
	}
	// Both writers use the same epoch/version/physical gate and complete ancestry.
	for name, create := range map[string]func(uint64, uint64) (scope.Chain, error){
		"school": func(epoch, version uint64) (scope.Chain, error) {
			return schools.Create(ctx, firstSchool, epoch, version)
		},
		"campus": func(epoch, version uint64) (scope.Chain, error) {
			return campuses.Create(ctx, firstCampus, epoch, version)
		},
	} {
		if _, err := create(2, version); !errors.Is(err, models.ErrConflict) {
			t.Fatal(name, "stale epoch", err)
		}
		if _, err := create(3, version-1); !errors.Is(err, models.ErrConflict) {
			t.Fatal(name, "stale version", err)
		}
		for _, state := range []string{"maintenance", "suspended", "retired", "provisioning"} {
			exec(p, "UPDATE tenant_gates SET state=? WHERE tenant_id=?", state, tenant.String())
			if _, err := create(3, version); !errors.Is(err, models.ErrConflict) {
				t.Fatal(name, state, err)
			}
		}
		exec(p, "UPDATE tenant_gates SET state='active',pool_id='wrong' WHERE tenant_id=?", tenant.String())
		if _, err := create(3, version); !errors.Is(err, models.ErrConflict) {
			t.Fatal(name, "physical gate", err)
		}
		exec(p, "UPDATE tenant_gates SET pool_id=(SELECT pool_id FROM edu_pool_marker WHERE singleton=1) WHERE tenant_id=?", tenant.String())
		exec(catalog.Control, "UPDATE edu_tenant_placement SET pool_id='wrong' WHERE tenant_id=?", tenant.String())
		if _, err := create(3, version); !errors.Is(err, database.ErrPlacement) {
			t.Fatal(name, "wrong pin", err)
		}
		var poolID string
		if err := p.DB.QueryRow("SELECT pool_id FROM edu_pool_marker WHERE singleton=1").Scan(&poolID); err != nil {
			t.Fatal(err)
		}
		exec(catalog.Control, "UPDATE edu_tenant_placement SET pool_id=? WHERE tenant_id=?", poolID, tenant.String())
	}
	for _, ancestor := range []scope.OrganizationID{root.ID(), child.ID()} {
		exec(p, "UPDATE tenancy_organizations SET active=false WHERE tenant_id=? AND id=?", tenant.String(), ancestor.String())
		if _, err := schools.Create(ctx, firstSchool, 3, version); !errors.Is(err, models.ErrScope) {
			t.Fatal("inactive organization", err)
		}
		exec(p, "UPDATE tenancy_organizations SET active=true WHERE tenant_id=? AND id=?", tenant.String(), ancestor.String())
	}
	// A real other-tenant parent must not be accepted, even within shared storage.
	other := requireValue(scope.NewID[scope.Tenant]())
	otherRoot := requireValue(scope.NewID[scope.Organization]())
	otherSchool := requireValue(scope.NewID[scope.School]())
	exec(p, "INSERT INTO tenant_gates (tenant_id,pool_id,placement_epoch,hierarchy_version,state) SELECT ?,pool_id,1,3,'active' FROM edu_pool_marker WHERE singleton=1", other.String())
	exec(p, "INSERT INTO tenancy_organizations (tenant_id,id,name,active) VALUES (?,?,'Other',true)", other.String(), otherRoot.String())
	exec(p, "INSERT INTO tenancy_schools (tenant_id,id,organization_id,code,name,active) VALUES (?,?,?,'MAIN','Other',true)", other.String(), otherSchool.String(), otherRoot.String())
	for _, parent := range []scope.OrganizationID{otherRoot, requireValue(scope.NewID[scope.Organization]())} {
		if _, err := schools.Create(ctx, school(parent, "cross"), 3, version); !errors.Is(err, models.ErrScope) {
			t.Fatal("missing/cross-tenant organization", err)
		}
	}
	for _, parent := range []scope.SchoolID{otherSchool, requireValue(scope.NewID[scope.School]())} {
		if _, err := campuses.Create(ctx, campus(parent, "cross"), 3, version); !errors.Is(err, models.ErrScope) {
			t.Fatal("missing/cross-tenant school", err)
		}
	}
	exec(p, "DELETE FROM tenancy_schools WHERE tenant_id=?", other.String())
	exec(p, "DELETE FROM tenancy_organizations WHERE tenant_id=?", other.String())
	exec(p, "DELETE FROM tenant_gates WHERE tenant_id=?", other.String())
	assertState(0, 0)
	// Fail after the insert/version update, then prove the same request can commit.
	exec(p, "ALTER TABLE edu_outbox RENAME COLUMN payload TO unavailable_payload")
	if _, err := schools.Create(ctx, firstSchool, 3, version); err == nil {
		t.Fatal("school without outbox")
	}
	exec(p, "ALTER TABLE edu_outbox RENAME COLUMN unavailable_payload TO payload")
	assertState(0, 0)
	chain, err := schools.Create(ctx, firstSchool, 3, version)
	if err != nil {
		t.Fatal(err)
	}
	version++
	checkChain(chain, "school", firstSchool.ID().String())
	if _, err := schools.Create(ctx, firstSchool, 3, version-1); !errors.Is(err, models.ErrConflict) {
		t.Fatal("school stale replay", err)
	}
	for _, candidate := range []models.School{firstSchool, school(child.ID(), "MAIN"), requireValue(models.NewSchool(firstSchool.ID(), tenant, root.ID(), "OTHER", "Duplicate identity"))} {
		if _, err := schools.Create(ctx, candidate, 3, version); err == nil {
			t.Fatal("duplicate school")
		}
	}
	for _, ancestor := range []scope.OrganizationID{root.ID(), child.ID()} {
		exec(p, "UPDATE tenancy_organizations SET active=false WHERE tenant_id=? AND id=?", tenant.String(), ancestor.String())
		if _, err := campuses.Create(ctx, firstCampus, 3, version); !errors.Is(err, models.ErrScope) {
			t.Fatal("campus inactive ancestor", err)
		}
		exec(p, "UPDATE tenancy_organizations SET active=true WHERE tenant_id=? AND id=?", tenant.String(), ancestor.String())
	}
	exec(p, "UPDATE tenancy_schools SET active=false WHERE tenant_id=? AND id=?", tenant.String(), firstSchool.ID().String())
	if _, err := campuses.Create(ctx, firstCampus, 3, version); !errors.Is(err, models.ErrScope) {
		t.Fatal("inactive school", err)
	}
	exec(p, "UPDATE tenancy_schools SET active=true WHERE tenant_id=? AND id=?", tenant.String(), firstSchool.ID().String())
	exec(p, "ALTER TABLE edu_outbox RENAME COLUMN payload TO unavailable_payload")
	if _, err := campuses.Create(ctx, firstCampus, 3, version); err == nil {
		t.Fatal("campus without outbox")
	}
	exec(p, "ALTER TABLE edu_outbox RENAME COLUMN unavailable_payload TO payload")
	assertState(1, 0)
	chain, err = campuses.Create(ctx, firstCampus, 3, version)
	if err != nil {
		t.Fatal(err)
	}
	version++
	checkChain(chain, "campus", firstCampus.ID().String())
	if _, err := campuses.Create(ctx, firstCampus, 3, version-1); !errors.Is(err, models.ErrConflict) {
		t.Fatal("campus stale replay", err)
	}
	for _, candidate := range []models.Campus{firstCampus, campus(firstSchool.ID(), "MAIN")} {
		if _, err := campuses.Create(ctx, candidate, 3, version); err == nil {
			t.Fatal("duplicate campus")
		}
	}
	// Codes are scoped to organization for schools and to school for campuses.
	secondSchool := school(root.ID(), "MAIN")
	if _, err := schools.Create(ctx, secondSchool, 3, version); err != nil {
		t.Fatal("same school code in another organization", err)
	}
	version++
	secondCampus := campus(secondSchool.ID(), "MAIN")
	if _, err := campuses.Create(ctx, secondCampus, 3, version); err != nil {
		t.Fatal("same campus code in another school", err)
	}
	version++
	if _, err := campuses.Create(ctx, requireValue(models.NewCampus(firstCampus.ID(), tenant, secondSchool.ID(), "OTHER", "Duplicate identity", "UTC")), 3, version); err == nil {
		t.Fatal("duplicate campus identity under another school")
	}
	if _, err := directory.Resolve(ctx, scope.Selection{Tenant: tenant, School: secondSchool.ID(), Campus: firstCampus.ID()}); !errors.Is(err, models.ErrScope) {
		t.Fatal("campus resolved through sibling school", err)
	}
	assertState(2, 2)
	// Same-kind and cross-kind writers compete on the single tenant version.
	for round, kinds := range [][2]string{{"school", "school"}, {"campus", "campus"}, {"school", "campus"}, {"organization", "campus"}} {
		done := make(chan error, 2)
		expected := version
		for i, kind := range kinds {
			go func() {
				var err error
				switch kind {
				case "school":
					_, err = schools.Create(ctx, school(child.ID(), fmt.Sprintf("race-%d-%d", round, i)), 3, expected)
				case "campus":
					_, err = campuses.Create(ctx, campus(firstSchool.ID(), fmt.Sprintf("race-%d-%d", round, i)), 3, expected)
				case "organization":
					_, err = orgs.Create(ctx, requireValue(models.NewOrganization(requireValue(scope.NewID[scope.Organization]()), tenant, root.ID(), "Concurrent")), 3, expected)
				}
				done <- err
			}()
		}
		first, second := <-done, <-done
		if !((first == nil && errors.Is(second, models.ErrConflict)) || (second == nil && errors.Is(first, models.ErrConflict))) {
			t.Fatal("concurrent outcomes", kinds, first, second)
		}
		version++
	}
	if got := scalar(p, "SELECT hierarchy_version FROM tenant_gates WHERE tenant_id=?", tenant.String()); got != int(version) {
		t.Fatal("final version", got, version)
	}
	if got := scalar(p, "SELECT count(*) FROM edu_outbox WHERE tenant_id=?", tenant.String()); got != int(version)-1 {
		t.Fatal("final event count", got, version)
	}
	rows := 0
	for _, table := range []string{"tenancy_organizations", "tenancy_schools", "tenancy_campuses"} {
		rows += scalar(p, "SELECT count(*) FROM "+table+" WHERE tenant_id=?", tenant.String())
	}
	if rows != int(version)-1 {
		t.Fatal("committed hierarchy row count", rows, version)
	}
	var code, name, zone string
	if err := p.DB.QueryRowContext(ctx, p.Bind("SELECT code,name FROM tenancy_schools WHERE tenant_id=? AND id=?"), tenant.String(), firstSchool.ID().String()).Scan(&code, &name); err != nil || code != "MAIN" || name != "School" {
		t.Fatal("canonical school", code, name, err)
	}
	if err := p.DB.QueryRowContext(ctx, p.Bind("SELECT code,name,timezone FROM tenancy_campuses WHERE tenant_id=? AND id=?"), tenant.String(), firstCampus.ID().String()).Scan(&code, &name, &zone); err != nil || code != "MAIN" || name != "Campus" || zone != "Africa/Nairobi" {
		t.Fatal("canonical campus", code, name, zone, err)
	}
	if p.DB != catalog.Control.DB {
		for _, table := range []string{"tenancy_organizations", "tenancy_schools", "tenancy_campuses", "tenant_gates", "edu_outbox"} {
			if n := scalar(catalog.Control, "SELECT count(*) FROM "+table+" WHERE tenant_id=?", tenant.String()); n != 0 {
				t.Fatal("dedicated data in shared pool", table, n)
			}
		}
	}
}
