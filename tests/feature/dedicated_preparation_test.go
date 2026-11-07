package feature

import (
	"errors"
	"testing"

	"github.com/elcomware/edupilot/app/provisioning"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/outbox"
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
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := target.DB.Exec(target.Bind(query), args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, owner := range []string{candidateID, "unexpected-owner"} {
		exec("INSERT INTO ws_tenancy (school_id,id,title,status,version) VALUES (?,'unexpected','fixture','open',1)", owner)
		check(operation, false)
		exec("DELETE FROM ws_tenancy WHERE school_id=? AND id='unexpected'", owner)
		check(operation, true)
	}
	exec("INSERT INTO tenancy_organizations (tenant_id,id,name,active) VALUES (?,'unexpected','fixture',true)", candidateID)
	check(operation, false)
	exec("DELETE FROM tenancy_organizations WHERE tenant_id=? AND id='unexpected'", candidateID)

	exec("INSERT INTO edu_outbox (event_id,tenant_id,event_type,aggregate_id,payload,created_at,available_at) VALUES ('unexpected',?,'fixture',?,'{}',0,0)", candidateID, candidateID)
	check(operation, false)
	exec("DELETE FROM edu_outbox WHERE event_id='unexpected'")
	processed, err := (outbox.Dispatcher{}).Step(ctx, target)
	if err != nil || !processed {
		t.Fatal("prepared event dispatch", processed, err)
	}
	check(operation, true)
	exec("UPDATE edu_audit_inbox SET tenant_id='unexpected-owner' WHERE event_id=?", operation)
	check(operation, false)
	exec("UPDATE edu_audit_inbox SET tenant_id=? WHERE event_id=?", candidateID, operation)
	check(operation, true)

	exec("INSERT INTO edu_audit_inbox (event_id,tenant_id,event_type,aggregate_id,received_at) VALUES ('unexpected',?,'fixture',?,0)", candidateID, candidateID)
	check(operation, false)
	exec("DELETE FROM edu_audit_inbox WHERE event_id='unexpected'")
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
