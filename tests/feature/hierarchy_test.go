package feature

import (
	"database/sql"
	"errors"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/elcomware/edupilot/modules/tenancy/app/services"
	"slices"
	"testing"

	"github.com/elcomware/edupilot/modules/core/database"
)

// Exercise the installed Blueprint schema with two synthetic tenants in a shared pool.
func verifyHierarchyConstraints(t *testing.T, p database.Pool) {
	t.Helper()
	const gate = "INSERT INTO tenant_gates (tenant_id,pool_id,placement_epoch,hierarchy_version,state) VALUES (?, 'pool', 1, 1, 'active')"
	const org = "INSERT INTO tenancy_organizations (tenant_id,id,parent_id,name,active) VALUES (?,?,?,'Group',true)"
	const school = "INSERT INTO tenancy_schools (tenant_id,id,organization_id,code,name,active) VALUES (?,?,?,?,'School',true)"
	const campus = "INSERT INTO tenancy_campuses (tenant_id,id,school_id,code,name,timezone,active) VALUES (?,?,?,?,'Campus','UTC',true)"
	insert := func(allowed bool, query string, args ...any) {
		t.Helper()
		_, err := p.DB.ExecContext(t.Context(), p.Bind(query), args...)
		if allowed && err != nil {
			t.Fatalf("valid hierarchy row %v: %v", args, err)
		}
		if !allowed && err == nil {
			t.Fatalf("accepted invalid hierarchy row %v", args)
		}
	}
	for _, tenant := range []string{"east", "west"} {
		insert(true, gate, tenant)
		insert(true, org, tenant, "root", nil)
		insert(true, school, tenant, "primary", "root", "PRI")
		insert(true, campus, tenant, "main", "primary", "MAIN")
	}
	insert(true, org, "east", "division", "root")
	insert(true, school, "east", "annex", "division", "PRI")
	insert(true, campus, "east", "annex-main", "annex", "MAIN")
	insert(false, org, "west", "child", "division")
	insert(false, school, "west", "annex", "division", "NEW")
	insert(false, campus, "west", "branch", "annex", "BRANCH")
	insert(false, school, "east", "duplicate", "division", "PRI")
	insert(false, campus, "east", "duplicate", "annex", "MAIN")
}

func verifyPersistedDirectory(t *testing.T, catalog *database.Catalog) {
	t.Helper()
	ctx := t.Context()
	directory := services.PersistedDirectory{Pools: catalog}
	tenant := requireValue(scope.NewID[scope.Tenant]())
	other := requireValue(scope.NewID[scope.Tenant]())
	root := requireValue(scope.NewID[scope.Organization]())
	div := requireValue(scope.NewID[scope.Organization]())
	school := requireValue(scope.NewID[scope.School]())
	campus := requireValue(scope.NewID[scope.Campus]())
	sel := scope.Selection{Tenant: tenant, Campus: campus}
	if _, err := directory.Resolve(ctx, sel); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unpinned lookup: %v", err)
	}
	var pins int
	if err := catalog.Control.DB.QueryRow(catalog.Control.Bind("SELECT count(*) FROM edu_tenant_placement WHERE tenant_id=?"), tenant.String()).Scan(&pins); err != nil || pins != 0 {
		t.Fatal("lookup registered a placement", pins, err)
	}
	p, err := catalog.ForTenantContext(ctx, tenant.String())
	if err != nil {
		t.Fatal(err)
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := p.DB.ExecContext(ctx, p.Bind(query), args...); err != nil {
			t.Fatal(err)
		}
	}
	const gate = "INSERT INTO tenant_gates (tenant_id,pool_id,placement_epoch,hierarchy_version,state) SELECT ?,pool_id,1,9,'active' FROM edu_pool_marker WHERE singleton=1"
	exec(gate, tenant.String())
	const org = "INSERT INTO tenancy_organizations (tenant_id,id,parent_id,name,active) VALUES (?,?,?,'Division',true)"
	exec(org, tenant.String(), root.String(), nil)
	exec(org, tenant.String(), div.String(), root.String())
	exec("INSERT INTO tenancy_schools (tenant_id,id,organization_id,code,name,active) VALUES (?,?,?,'PRI','Primary',true)", tenant.String(), school.String(), div.String())
	exec("INSERT INTO tenancy_campuses (tenant_id,id,school_id,code,name,timezone,active) VALUES (?,?,?,'MAIN','Main','UTC',true)", tenant.String(), campus.String(), school.String())
	chain, err := directory.Resolve(ctx, sel)
	if err != nil || !chain.Active || chain.HierarchyVersion != 9 || chain.Tenant != tenant || chain.School != school || chain.Organization != div || !slices.Equal(chain.Ancestors, []scope.OrganizationID{root, div}) {
		t.Fatal(chain, err)
	}
	mismatch := sel
	mismatch.Organization = root
	if _, err := directory.Resolve(ctx, mismatch); !errors.Is(err, scope.ErrUnknown) {
		t.Fatal("accepted conflicting ancestor", err)
	}
	if _, err := catalog.ForTenantContext(ctx, other.String()); err != nil {
		t.Fatal(err)
	}
	exec(gate, other.String())
	if _, err := directory.Resolve(ctx, scope.Selection{Tenant: other, Campus: campus}); !errors.Is(err, scope.ErrUnknown) {
		t.Fatal("read another tenant's campus", err)
	}
	exec("UPDATE tenancy_organizations SET active=false WHERE tenant_id=? AND id=?", tenant.String(), root.String())
	chain, err = directory.Resolve(ctx, sel)
	if err != nil || chain.Active {
		t.Fatal("inactive ancestor", chain, err)
	}
	exec("UPDATE tenancy_organizations SET parent_id=? WHERE tenant_id=? AND id=?", div.String(), tenant.String(), root.String())
	if _, err := directory.Resolve(ctx, sel); !errors.Is(err, scope.ErrUnknown) {
		t.Fatal("accepted cycle", err)
	}
	exec("UPDATE tenant_gates SET pool_id='wrong' WHERE tenant_id=?", tenant.String())
	if _, err := directory.Resolve(ctx, scope.Selection{Tenant: tenant}); !errors.Is(err, scope.ErrUnknown) {
		t.Fatal("accepted incorrect gate", err)
	}
}

func requireValue[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}
