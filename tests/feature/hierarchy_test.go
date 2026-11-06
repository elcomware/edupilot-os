package feature

import (
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
