package edumigrations_test

import (
	"strings"

	"testing"

	"github.com/elcomware/edupilot/platform/database"
)

// Exercise the same owner/pool constraints on both supported SQL engines.
func TestFoundationPlacementOwnership(t *testing.T) {
	for _, driver := range []string{"sqlite", "postgres"} {
		t.Run(driver, func(t *testing.T) {
			pool := foundationSQL(t, driver)
			db := pool.DB
			for _, stmt := range database.FoundationControlDDL(driver) {
				if _, err := db.Exec(stmt); err != nil {
					t.Fatal(err)
				}
			}
			exec := func(wantOK bool, query string, args ...any) {
				t.Helper()
				_, err := db.Exec(pool.Bind(query), args...)
				if (err == nil) != wantOK {
					t.Fatalf("statement success should be %v: %v", wantOK, err)
				}
			}
			north := "04557827-25a7-46d8-befa-8a07a9b84b58"
			west := "b776537e-3b1f-4dca-9793-2f25814868eb"
			for _, tenant := range []string{north, west} {
				exec(true, "INSERT INTO tenants (id,slug,state,version) VALUES (?,?,'active',1)", tenant, tenant)
			}
			shared := "41eeed8a-a35c-4f41-8a4c-a4fbec455168"
			dedicated := "276ca167-4d65-44cd-a6b7-b0c94a6ca748"
			addPool := "INSERT INTO database_pools (id,connection_name,pool_marker,driver,kind,dedicated_tenant_id) VALUES (?,?,?,?,?,?)"
			exec(true, addPool, shared, "shared_config", "marker-A", driver, "shared", nil)
			exec(true, addPool, dedicated, "north_config", "marker-B", driver, "dedicated", north)
			exec(false, addPool, "b8dd054f-ddf3-4b44-8369-b818f36da45a", "wrong_config", "marker-C", driver, "dedicated", nil)
			add := "INSERT INTO tenant_placements (tenant_id,pool_id,pool_kind,dedicated_owner,epoch,state) VALUES (?,?,?,?,?,?)"
			exec(false, add, west, dedicated, "dedicated", west, 1, "active")
			exec(false, add, west, dedicated, "dedicated", nil, 1, "active")
			exec(false, add, west, dedicated, "shared", nil, 1, "active")
			exec(true, add, north, dedicated, "dedicated", north, 1, "active")
			exec(false, add, north, shared, "shared", nil, 2, "active")
			exec(false, add, west, shared, "shared", nil, 0, "active")
			exec(true, add, west, shared, "shared", nil, 1, "active")
			exec(false, "UPDATE database_pools SET pool_marker='replacement' WHERE id=?", shared)
			exec(false, "UPDATE tenant_placements SET epoch=? WHERE tenant_id=?", "bogus", west)
			exec(true, "DELETE FROM tenant_placements WHERE tenant_id=?", north)
			exec(false, "UPDATE database_pools SET dedicated_tenant_id=? WHERE id=?", west, dedicated)
			exec(false, "DELETE FROM database_pools WHERE id=?", dedicated)
			if driver == "sqlite" {
				var hidden int64
				if err := db.QueryRow("SELECT rowid FROM database_pools WHERE id=?", dedicated).Scan(&hidden); err == nil {
					// Even entirely fresh public keys can replace an unreferenced pool via a hidden ROWID.
					exec(false, "INSERT OR REPLACE INTO database_pools(rowid,id,connection_name,pool_marker,driver,kind) VALUES (?,?,'fresh','fresh-marker','sqlite','shared')", hidden, orgA)
				}
				replace := strings.Replace(addPool, "INSERT INTO", "INSERT OR REPLACE INTO", 1)
				exec(false, replace, dedicated, "north_config", "marker-B", driver, "dedicated", west)
				exec(false, replace, orgA, "north_config", "marker-D", driver, "shared", nil)
				exec(false, replace, orgA, "alt_config", "marker-B", driver, "shared", nil)
			}
			run := north
			exec(true, "INSERT INTO tenant_upgrade_runs(id,manifest_version,manifest_sha256,source_sha256,stage,revision,created_at,updated_at) VALUES (?,1,?,?,'prepared',1,0,0)", run, strings.Repeat("a", 64), strings.Repeat("b", 64))
			mapping := "INSERT INTO legacy_scope_mappings(legacy_school_id,run_id,tenant_id,source_pool_id,source_marker,organization_id,school_id,campus_id) VALUES (?,?,?,?,?,?,?,?)"
			exec(false, mapping, "legacy-a", run, north, shared, "marker-B", groupA, schoolA, orgA)
			exec(true, mapping, "legacy-a", run, north, dedicated, "marker-B", groupA, schoolA, orgA)
			exec(false, mapping, "legacy-b", run, north, shared, "marker-A", groupB, schoolB, orgB)

		})
	}
}
