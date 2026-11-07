package feature

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/elcomware/edupilot/app/provisioning"
	auth "github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/outbox"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/elcomware/edupilot/modules/tenancy/app/models"
	"github.com/elcomware/edupilot/modules/tenancy/app/services"
)

func verifyDedicatedActivation(t *testing.T, db *database.Catalog) {
	t.Helper()
	ctx := t.Context()
	tenant := requireValue(scope.Parse[scope.Tenant](candidateID))
	target := db.Named()["tenant_"+candidateID]
	var pool, operation string
	if err := db.Control.DB.QueryRow(db.Control.Bind("SELECT pool_id,operation_id FROM edu_tenant_reservations WHERE tenant_id=?"), candidateID).Scan(&pool, &operation); err != nil {
		t.Fatal(err)
	}
	exec := func(p database.Pool, q string, args ...any) {
		t.Helper()
		if _, err := p.DB.Exec(p.Bind(q), args...); err != nil {
			t.Fatal(err)
		}
	}
	check := func(op string, want bool) provisioning.DedicatedActivation {
		t.Helper()
		receipt, err := provisioning.ActivateDedicatedTenant(ctx, db, tenant, pool, op)
		if (err == nil) != want {
			t.Fatalf("activate %+v: %v", receipt, err)
		}
		return receipt
	}
	count := func(p database.Pool, q string, want int) {
		t.Helper()
		var got int
		if err := p.DB.QueryRow(q).Scan(&got); err != nil || got != want {
			t.Fatal(q, got, want, err)
		}
	}
	verifyActivationRealmFence(t, db, tenant, pool, operation)
	check("wrong", false)
	if _, err := provisioning.ActivateDedicatedTenant(ctx, db, tenant, "wrong", operation); err == nil {
		t.Fatal("accepted wrong pool")
	}
	for _, fixture := range []struct {
		p              database.Pool
		insert, remove string
	}{
		{db.Control, "INSERT INTO auth_roles (school_id,name) VALUES (?,'unexpected')", "DELETE FROM auth_roles WHERE school_id=? AND name='unexpected'"},
		{db.Control, "INSERT INTO auth_permissions (school_id,name) VALUES (?,'unexpected')", "DELETE FROM auth_permissions WHERE school_id=? AND name='unexpected'"},
		{db.Control, "INSERT INTO auth_users (school_id,id,email,password_hash) VALUES (?,'unexpected','fixture@example.test','invalid')", "DELETE FROM auth_users WHERE school_id=? AND id='unexpected'"},
		{target, "INSERT INTO ws_tenancy (school_id,id,title,status,version) VALUES (?,'unexpected','fixture','open',1)", "DELETE FROM ws_tenancy WHERE school_id=? AND id='unexpected'"},
		{db.Control, "INSERT INTO ws_tenancy (school_id,id,title,status,version) VALUES (?,'unexpected','fixture','open',1)", "DELETE FROM ws_tenancy WHERE school_id=? AND id='unexpected'"},
	} {
		exec(fixture.p, fixture.insert, candidateID)
		check(operation, false)
		exec(fixture.p, fixture.remove, candidateID)
	}
	exec(target, "INSERT INTO ws_tenancy (school_id,id,title,status,version) VALUES ('other-owner','unexpected','fixture','open',1)")
	check(operation, false)
	exec(target, "DELETE FROM ws_tenancy WHERE school_id='other-owner'")
	exec(target, "UPDATE tenant_gates SET hierarchy_version=2 WHERE tenant_id=?", candidateID)
	check(operation, false)
	exec(target, "UPDATE tenant_gates SET hierarchy_version=1 WHERE tenant_id=?", candidateID)
	exec(target, "INSERT INTO edu_outbox (event_id,tenant_id,event_type,aggregate_id,payload,created_at,available_at) VALUES ('unexpected',?,'fixture',?,'{}',0,0)", candidateID, candidateID)
	check(operation, false)
	exec(target, "DELETE FROM edu_outbox WHERE event_id='unexpected'")
	exec(target, "INSERT INTO edu_audit_inbox (event_id,tenant_id,event_type,aggregate_id,received_at) VALUES ('unexpected',?,'fixture',?,0)", candidateID, candidateID)
	check(operation, false)
	exec(target, "DELETE FROM edu_audit_inbox WHERE event_id='unexpected'")
	var preparedPayload string
	if err := target.DB.QueryRow(target.Bind("SELECT payload FROM edu_outbox WHERE event_id=?"), operation).Scan(&preparedPayload); err != nil {
		t.Fatal(err)
	}
	exec(target, "UPDATE edu_outbox SET payload='{}' WHERE event_id=?", operation)
	check(operation, false)
	exec(target, "UPDATE edu_outbox SET payload=? WHERE event_id=?", preparedPayload, operation)
	// An event insert failure must roll the gate transition back.
	failInsert(t, target, "edu_outbox", true)
	check(operation, false)
	count(target, "SELECT count(*) FROM tenant_gates WHERE state='provisioning'", 1)
	count(target, "SELECT count(*) FROM edu_outbox", 1)
	failInsert(t, target, "edu_outbox", false)
	// A control failure after target commit preserves the pending routing fence.
	failInsert(t, db.Control, "edu_tenant_activations", true)
	activationResults := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := provisioning.ActivateDedicatedTenant(ctx, db, tenant, pool, operation)
			activationResults <- err
		}()
	}
	firstActivation, secondActivation := <-activationResults, <-activationResults
	if firstActivation == nil || secondActivation == nil {
		t.Fatalf("activation published through failing receipt insert: %v, %v", firstActivation, secondActivation)
	}
	if !strings.Contains(firstActivation.Error(), "injected activation failure") && !strings.Contains(secondActivation.Error(), "injected activation failure") {
		t.Fatalf("neither activation reached the injected publication failure: %v, %v", firstActivation, secondActivation)
	}
	t.Logf("activation publication failures: %v, %v", firstActivation, secondActivation)
	count(target, "SELECT count(*) FROM tenant_gates WHERE state='active'", 1)
	count(target, "SELECT count(*) FROM edu_outbox", 2)
	count(db.Control, "SELECT count(*) FROM edu_tenant_activations", 0)
	if _, err := db.ForTenantContext(ctx, candidateID); err == nil {
		t.Fatal("pending activation routed")
	}
	if _, err := db.ForTenantContext(ctx, "west"); err != nil {
		t.Fatal("unrelated tenant unavailable", err)
	}
	check(operation, false)
	// Recovery must revalidate both databases even after the target is active.
	exec(db.Control, "INSERT INTO auth_roles (school_id,name) VALUES (?,'recovery-drift')", candidateID)
	check(operation, false)
	exec(db.Control, "DELETE FROM auth_roles WHERE school_id=? AND name='recovery-drift'", candidateID)
	exec(target, "INSERT INTO ws_tenancy (school_id,id,title,status,version) VALUES ('other-owner','recovery-drift','fixture','open',1)")
	check(operation, false)
	exec(target, "DELETE FROM ws_tenancy WHERE school_id='other-owner'")
	processed, err := (outbox.Dispatcher{}).Step(ctx, target)
	if err != nil || !processed {
		t.Fatal("activation event dispatch", processed, err)
	}
	var activationID, activationPayload string
	if err := target.DB.QueryRow("SELECT event_id,payload FROM edu_outbox WHERE event_type='tenancy.tenant.activated.v1'").Scan(&activationID, &activationPayload); err != nil {
		t.Fatal(err)
	}
	exec(target, "UPDATE edu_audit_inbox SET tenant_id='corrupt' WHERE event_id=?", activationID)
	check(operation, false)
	exec(target, "UPDATE edu_audit_inbox SET tenant_id=? WHERE event_id=?", candidateID, activationID)
	exec(target, "UPDATE edu_outbox SET payload='{}' WHERE event_id=?", activationID)
	check(operation, false)
	exec(target, "UPDATE edu_outbox SET payload=? WHERE event_id=?", activationPayload, activationID)
	failInsert(t, db.Control, "edu_tenant_activations", false)
	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := provisioning.ActivateDedicatedTenant(ctx, db, tenant, pool, operation)
			results <- err
		}()
	}
	first, second := <-results, <-results
	if first != nil && second != nil {
		t.Fatal("both activations failed", first, second)
	}
	receipt := check(operation, true)
	repeat := check(operation, true)
	before, _ := json.Marshal(receipt)
	after, _ := json.Marshal(repeat)
	if string(before) != string(after) || receipt.OperationID != operation || receipt.PoolID != pool || receipt.Tenant != tenant || receipt.CompletedAt == "" {
		t.Fatal("unstable receipt", receipt, repeat)
	}
	count(target, "SELECT count(*) FROM edu_outbox", 2)
	count(db.Control, "SELECT count(*) FROM edu_tenant_activations", 1)
	count(db.Control, "SELECT count(*) FROM edu_tenant_reservations WHERE tenant_id='"+candidateID+"'", 0)
	routed, err := db.PinnedPool(ctx, candidateID)
	if err != nil || routed.DB != target.DB {
		t.Fatal("dedicated routing", err)
	}
	if _, err := auth.New(&db.Control).Seed(ctx, candidateID, "admin@example.test", "synthetic-password-long"); err == nil {
		t.Fatal("activated realm bootstrapped")
	}
	check("wrong", false)
	organization := requireValue(models.NewOrganization(requireValue(scope.NewID[scope.Organization]()), tenant, scope.OrganizationID{}, "First organization"))
	if _, err := (services.OrganizationWriter{Pools: db}).Create(ctx, organization, 1, 1); err != nil {
		t.Fatal(err)
	}
	repeat = check(operation, true)
	after, _ = json.Marshal(repeat)
	if string(before) != string(after) {
		t.Fatal("hierarchy edit changed receipt")
	}
	exec(db.Control, "UPDATE edu_tenant_activations SET payload='{}' WHERE tenant_id=?", candidateID)
	check(operation, false)
	exec(db.Control, "UPDATE edu_tenant_activations SET payload=? WHERE tenant_id=?", string(before), candidateID)
	exec(target, "UPDATE edu_outbox SET payload='{}' WHERE event_id=?", receipt.EventID)
	check(operation, false)
	exec(target, "UPDATE edu_outbox SET payload=? WHERE event_id=?", string(before), receipt.EventID)
	check(operation, true)
	if err := db.CheckReady(ctx); err != nil {
		t.Fatal(err)
	}
}

func failInsert(t *testing.T, p database.Pool, table string, enable bool) {
	t.Helper()
	var queries []string
	if p.Driver == "postgres" {
		if enable {
			queries = []string{"CREATE FUNCTION activation_fail_insert() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected activation failure'; END $$", "CREATE TRIGGER activation_fail_insert BEFORE INSERT ON " + table + " FOR EACH ROW EXECUTE FUNCTION activation_fail_insert()"}
		} else {
			queries = []string{"DROP TRIGGER activation_fail_insert ON " + table, "DROP FUNCTION activation_fail_insert()"}
		}
	} else if enable {
		queries = []string{"CREATE TRIGGER activation_fail_insert BEFORE INSERT ON " + table + " BEGIN SELECT RAISE(ABORT,'injected activation failure'); END"}
	} else {
		queries = []string{"DROP TRIGGER activation_fail_insert"}
	}
	for _, q := range queries {
		if _, err := p.DB.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
}

// The competing enrollment insert holds a realm FK lock before activation can
// acquire its fence. PostgreSQL must re-read after that writer commits.
func verifyActivationRealmFence(t *testing.T, db *database.Catalog, tenant scope.TenantID, pool, operation string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	writer, err := db.Control.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Rollback()
	if _, err := writer.ExecContext(ctx, db.Control.Bind("INSERT INTO auth_roles (school_id,name) VALUES (?,'concurrent-drift')"), tenant.String()); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := provisioning.ActivateDedicatedTenant(ctx, db, tenant, pool, operation)
		result <- err
	}()
	if db.Control.Driver == "postgres" {
		deadline := time.Now().Add(5 * time.Second)
		for {
			var blocked bool
			err := db.Control.DB.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE 'SELECT school_id FROM auth_realms%FOR UPDATE')`).Scan(&blocked)
			if err != nil {
				t.Fatal(err)
			}
			if blocked {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("activation never waited for realm lock")
			}
			time.Sleep(10 * time.Millisecond)
		}
	} else {
		select {
		case err := <-result:
			t.Fatal("activation bypassed pending writer", err)
		case <-time.After(50 * time.Millisecond):
		}
	}
	if err := writer.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-result; err == nil {
		t.Fatal("activation missed enrollment committed before realm fence")
	}
	if _, err := db.Control.DB.Exec(db.Control.Bind("DELETE FROM auth_roles WHERE school_id=? AND name='concurrent-drift'"), tenant.String()); err != nil {
		t.Fatal(err)
	}
	// Once acquired, the Auth fence excludes a new enrollment root until release.
	lock, err := db.Control.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Rollback()
	if err := auth.New(&db.Control).LockEmptyRealm(ctx, lock, tenant.String()); err != nil {
		t.Fatal(err)
	}
	writerCtx, writerCancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer writerCancel()
	_, err = db.Control.DB.ExecContext(writerCtx, db.Control.Bind("INSERT INTO auth_roles (school_id,name) VALUES (?,'blocked-drift')"), tenant.String())
	if err == nil {
		t.Fatal("enrollment bypassed held realm fence")
	}
	if err := lock.Rollback(); err != nil {
		t.Fatal(err)
	}
}
