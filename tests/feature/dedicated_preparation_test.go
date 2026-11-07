package feature

import (
	"errors"
	"testing"

	"github.com/elcomware/edupilot/app/provisioning"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
)

func verifyDedicatedPreparation(t *testing.T, db *database.Catalog) {
	t.Helper()
	ctx := t.Context()
	tenant := requireValue(scope.Parse[scope.Tenant](candidateID))
	var pool, operation string
	if err := db.Control.DB.QueryRow(db.Control.Bind("SELECT pool_id,operation_id FROM edu_tenant_reservations WHERE tenant_id=?"), candidateID).Scan(&pool, &operation); err != nil {
		t.Fatal(err)
	}
	target := db.Named()["tenant_"+candidateID]
	check := func(op string, want bool) {
		t.Helper()
		report, err := provisioning.PrepareDedicatedTenant(ctx, db, tenant, pool, op)
		if (err == nil) != want || (want && (report.Tenant != tenant || report.Active)) {
			t.Fatalf("prepare: %+v %v", report, err)
		}
	}
	check("wrong", false)
	if _, err := target.DB.Exec("ALTER TABLE edu_outbox RENAME COLUMN payload TO unavailable_payload"); err != nil {
		t.Fatal(err)
	}
	check(operation, false)
	var count int
	if err := target.DB.QueryRow(target.Bind("SELECT count(*) FROM tenant_gates WHERE tenant_id=?"), candidateID).Scan(&count); err != nil || count != 0 {
		t.Fatal("failed preparation left a gate", count, err)
	}
	if _, err := target.DB.Exec("ALTER TABLE edu_outbox RENAME COLUMN unavailable_payload TO payload"); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := provisioning.PrepareDedicatedTenant(ctx, db, tenant, pool, operation)
			results <- err
		}()
	}
	first, second := <-results, <-results
	if first != nil && second != nil {
		t.Fatal("both preparations failed", first, second)
	}
	check(operation, true)
	check(operation, true)
	var gates, events int
	if err := target.DB.QueryRow(target.Bind("SELECT count(*) FROM tenant_gates WHERE tenant_id=? AND state='provisioning' AND placement_epoch=1 AND hierarchy_version=1"), candidateID).Scan(&gates); err != nil || gates != 1 {
		t.Fatal(gates, err)
	}
	if err := target.DB.QueryRow(target.Bind("SELECT count(*) FROM edu_outbox WHERE tenant_id=?"), candidateID).Scan(&events); err != nil || events != 1 {
		t.Fatal(events, err)
	}
	if _, err := db.ForTenantContext(ctx, candidateID); !errors.Is(err, database.ErrPlacement) {
		t.Fatal("prepared tenant routed", err)
	}
	if err := db.CheckReady(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := target.DB.Exec(target.Bind("UPDATE tenant_gates SET state='active' WHERE tenant_id=?"), candidateID); err != nil {
		t.Fatal(err)
	}
	check(operation, false)
	if _, err := target.DB.Exec(target.Bind("UPDATE tenant_gates SET state='provisioning' WHERE tenant_id=?"), candidateID); err != nil {
		t.Fatal(err)
	}
	if _, err := target.DB.Exec(target.Bind("UPDATE edu_outbox SET payload='{}' WHERE tenant_id=?"), candidateID); err != nil {
		t.Fatal(err)
	}
	check(operation, false)
}
