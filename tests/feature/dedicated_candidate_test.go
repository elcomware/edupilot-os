package feature

import (
	"testing"

	"github.com/elcomware/edupilot/app/provisioning"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
)

const candidateID = "6e622843-7fe6-4a9f-89f8-834573c034ee"

func verifyDedicatedCandidate(t *testing.T, db *database.Catalog) {
	t.Helper()
	ctx := t.Context()
	id := requireValue(scope.Parse[scope.Tenant](candidateID))
	target := db.Named()["tenant_"+candidateID]
	var marker string
	if err := target.DB.QueryRow("SELECT pool_id FROM edu_pool_marker WHERE singleton=1").Scan(&marker); err != nil {
		t.Fatal(err)
	}
	check := func(poolID string, want bool) {
		t.Helper()
		result, err := provisioning.CheckDedicatedTenant(ctx, db, id, poolID)
		if (err == nil) != want || (want && (result.Tenant != id || result.PoolID != poolID)) {
			t.Fatalf("unexpected dedicated preflight: %+v, %v", result, err)
		}
	}
	check("wrong", false)
	check(marker, true)
	for _, fixture := range []struct {
		pool  database.Pool
		owner string
	}{{target, "another-school"}, {db.Named()["tenant_west"], candidateID}, {db.Control, candidateID}} {
		if _, err := fixture.pool.DB.Exec(fixture.pool.Bind("INSERT INTO ws_tenancy (school_id,id,title,status,version) VALUES (?, 'proof','fixture','open',1)"), fixture.owner); err != nil {
			t.Fatal(err)
		}
		check(marker, false)
		if _, err := fixture.pool.DB.Exec(fixture.pool.Bind("DELETE FROM ws_tenancy WHERE school_id=? AND id='proof'"), fixture.owner); err != nil {
			t.Fatal(err)
		}
	}
	check(marker, true)
	var n int
	if err := db.Control.DB.QueryRow(db.Control.Bind("SELECT count(*) FROM edu_tenant_placement WHERE tenant_id=?"), candidateID).Scan(&n); err != nil || n != 0 {
		t.Fatal("preflight created a placement", n, err)
	}
	if _, err := db.ForTenantContext(ctx, candidateID); err != nil {
		t.Fatal(err)
	}
	check(marker, false)
}
