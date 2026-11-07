package feature

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/app/provisioning"
	auth "github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/outbox"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func verifyDedicatedEnrollment(t *testing.T, db *database.Catalog) {
	t.Helper()
	ctx := t.Context()
	tenant := requireValue(scope.Parse[scope.Tenant](candidateID))
	control, target := db.Control, db.Named()["tenant_"+candidateID]
	var pool, operation string
	if err := control.DB.QueryRow(control.Bind("SELECT pool_id,operation_id FROM edu_tenant_activations WHERE tenant_id=?"), candidateID).Scan(&pool, &operation); err != nil {
		t.Fatal(err)
	}
	const password = "synthetic-password-long"
	exec := func(p database.Pool, q string, args ...any) {
		t.Helper()
		if _, err := p.DB.Exec(p.Bind(q), args...); err != nil {
			t.Fatal(err)
		}
	}
	scalar := func(p database.Pool, q string, args ...any) string {
		t.Helper()
		var value string
		if err := p.DB.QueryRow(p.Bind(q), args...).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	count := func(table string, want string) {
		t.Helper()
		if got := scalar(control, "SELECT count(*) FROM "+table+" WHERE school_id=?", candidateID); got != want {
			t.Fatalf("%s: got %s want %s", table, got, want)
		}
	}
	check := func(email, pw string, want bool) auth.AdminEnrollment {
		t.Helper()
		receipt, err := provisioning.EnrollDedicatedAdmin(ctx, db, tenant, pool, operation, email, pw)
		if (err == nil) != want {
			t.Fatalf("enrollment success=%v wanted=%v error=%v", err == nil, want, err)
		}
		return receipt
	}
	deny := func() { t.Helper(); check("admin@example.test", password, false) }
	if _, err := provisioning.EnrollDedicatedAdmin(ctx, db, tenant, pool, uuid.NewString(), "admin@example.test", password); err == nil {
		t.Fatal("wrong operation accepted")
	}
	if _, err := provisioning.EnrollDedicatedAdmin(ctx, db, tenant, "wrong", operation, "admin@example.test", password); err == nil {
		t.Fatal("wrong pool accepted")
	}
	if _, err := provisioning.EnrollDedicatedAdmin(ctx, &database.Catalog{Control: control}, tenant, pool, operation, "admin@example.test", password); err == nil {
		t.Fatal("missing dedicated route accepted")
	}
	check("invalid", password, false)
	check("admin@example.test", "short", false)
	eventID := uuid.NewSHA1(uuid.MustParse(operation), []byte(auth.AdminEnrollmentEvent)).String()
	// The absence of an enrollment receipt never permits adoption of orphan Auth
	// or audit data. Every failing initial attempt leaves zero administrators.
	for _, fixture := range []struct {
		insert, remove string
		args           []any
	}{
		{"INSERT INTO auth_roles (school_id,name) VALUES (?,'orphan')", "DELETE FROM auth_roles WHERE school_id=?", []any{candidateID}},
		{"INSERT INTO auth_permissions (school_id,name) VALUES (?,'orphan')", "DELETE FROM auth_permissions WHERE school_id=?", []any{candidateID}},
		{"INSERT INTO auth_users (school_id,id,email,password_hash) VALUES (?,'orphan','orphan@example.test','invalid')", "DELETE FROM auth_users WHERE school_id=?", []any{candidateID}},
		{"INSERT INTO auth_admin_enrollments (school_id,operation_id,payload,state_digest) VALUES (?,?,'{}','invalid')", "DELETE FROM auth_admin_enrollments WHERE school_id=?", []any{candidateID, operation}},
		{"INSERT INTO edu_audit_inbox (event_id,tenant_id,event_type,aggregate_id,received_at) VALUES (?,?,'auth.admin.enrolled.v1',?,0)", "DELETE FROM edu_audit_inbox WHERE tenant_id=?", []any{eventID, candidateID, candidateID}},
		{"INSERT INTO edu_outbox (event_id,tenant_id,event_type,aggregate_id,payload,created_at,available_at) VALUES (?,?,'auth.admin.enrolled.v1',?,'{}',0,0)", "DELETE FROM edu_outbox WHERE tenant_id=?", []any{eventID, candidateID, candidateID}},
	} {
		exec(control, fixture.insert, fixture.args...)
		deny()
		if strings.Contains(fixture.insert, "INSERT INTO auth_admin_enrollments") {
			if _, err := provisioning.ActivateDedicatedTenant(ctx, db, tenant, pool, operation); err == nil {
				t.Fatal("activation accepted orphan Auth receipt")
			}
		}
		exec(control, fixture.remove, candidateID)
		count("auth_users", "0")
	}
	exec(control, "INSERT INTO edu_tenant_reservations (tenant_id,operation_id,pool_id,created_at) VALUES (?,?,?,'2026-11-07T00:00:00Z')", candidateID, operation, pool)
	deny()
	exec(control, "DELETE FROM edu_tenant_reservations WHERE tenant_id=?", candidateID)
	exec(control, "DELETE FROM auth_realms WHERE school_id=?", candidateID)
	deny()
	exec(control, "INSERT INTO auth_realms (school_id) VALUES (?)", candidateID)
	// Check receipt, route, owner, gate and exact historical events before any Auth writes.
	for _, fixture := range []struct {
		p                           database.Pool
		table, column, where, value string
		args                        []any
	}{
		{control, "edu_tenant_activations", "payload", "tenant_id=?", "{}", []any{candidateID}},
		{control, "edu_tenant_placement", "pool_id", "tenant_id=?", "wrong", []any{candidateID}},
		{target, "edu_pool_marker", "owner", "singleton=1", "tenant:wrong", nil},
		{target, "edu_pool_marker", "pool_id", "singleton=1", "wrong", nil},
		{target, "tenant_gates", "state", "tenant_id=?", "suspended", []any{candidateID}},
		{target, "tenant_gates", "pool_id", "tenant_id=?", "wrong", []any{candidateID}},
		{target, "tenant_gates", "placement_epoch", "tenant_id=?", "2", []any{candidateID}},
		{target, "edu_outbox", "payload", "event_id=?", "{}", []any{operation}},
		{target, "edu_outbox", "payload", "event_type='tenancy.tenant.activated.v1'", "{}", nil},
	} {
		original := scalar(fixture.p, "SELECT "+fixture.column+" FROM "+fixture.table+" WHERE "+fixture.where, fixture.args...)
		query := "UPDATE " + fixture.table + " SET " + fixture.column + "=? WHERE " + fixture.where
		exec(fixture.p, query, append([]any{fixture.value}, fixture.args...)...)
		deny()
		exec(fixture.p, query, append([]any{original}, fixture.args...)...)
		count("auth_users", "0")
	}
	for _, p := range []database.Pool{control, db.Named()["tenant_west"]} {
		exec(p, "INSERT INTO ws_tenancy (school_id,id,title,status,version) VALUES (?,'unexpected','fixture','open',1)", candidateID)
		deny()
		exec(p, "DELETE FROM ws_tenancy WHERE school_id=? AND id='unexpected'", candidateID)
	}
	exec(target, "INSERT INTO ws_tenancy (school_id,id,title,status,version) VALUES ('foreign','unexpected','fixture','open',1)")
	deny()
	exec(target, "DELETE FROM ws_tenancy WHERE school_id='foreign'")
	// Both failures occur after create() has inserted the real user and grants.
	for _, table := range []string{"auth_admin_enrollments", "edu_outbox"} {
		failInsert(t, control, table, true)
		if _, err := provisioning.EnrollDedicatedAdmin(ctx, db, tenant, pool, operation, "admin@example.test", password); err == nil || !strings.Contains(err.Error(), "injected activation failure") {
			t.Fatalf("enrollment did not reach injected failure after Auth creation: %v", err)
		}
		failInsert(t, control, table, false)
		for _, authTable := range []string{"auth_users", "auth_roles", "auth_permissions", "auth_role_permissions", "auth_memberships", "auth_admin_enrollments"} {
			count(authTable, "0")
		}
		if got := scalar(control, "SELECT count(*) FROM edu_outbox WHERE tenant_id=?", candidateID); got != "0" {
			t.Fatal("failed enrollment left an event")
		}
	}
	// Independent credentials race for the same reserved realm. Exactly one wins.
	type result struct {
		email   string
		receipt auth.AdminEnrollment
		err     error
	}
	results := make(chan result, 2)
	for _, email := range []string{"admin@example.test", "competing@example.test"} {
		go func() {
			r, e := provisioning.EnrollDedicatedAdmin(ctx, db, tenant, pool, operation, email, password)
			results <- result{email, r, e}
		}()
	}
	first, second := <-results, <-results
	if (first.err == nil) == (second.err == nil) {
		t.Fatalf("want one enrollment winner: %v, %v", first.err, second.err)
	}
	winner := first
	if winner.err != nil {
		winner = second
	}
	email, receipt := winner.email, winner.receipt
	count("auth_users", "1")
	count("auth_admin_enrollments", "1")
	retry := check(strings.ToUpper(email), password, true)
	before, _ := json.Marshal(receipt)
	after, _ := json.Marshal(retry)
	if string(before) != string(after) || receipt.UserID == "" || receipt.EventID != eventID {
		t.Fatal("unstable enrollment receipt")
	}
	if strings.Contains(string(before), email) || strings.Contains(string(before), password) {
		t.Fatal("credential in enrollment receipt")
	}
	for _, wrong := range []struct{ email, password string }{{"other@example.test", password}, {email, "different-password-long"}} {
		check(wrong.email, wrong.password, false)
	}
	if _, err := provisioning.ActivateDedicatedTenant(ctx, db, tenant, pool, operation); err == nil {
		t.Fatal("activation retry accepted populated Auth")
	}
	// Delivery puts the sole permitted audit receipt beside the enrollment event.
	for i := 0; i < 100; i++ {
		if scalar(control, "SELECT count(*) FROM edu_audit_inbox WHERE event_id=?", eventID) == "1" {
			break
		}
		done, err := (outbox.Dispatcher{}).Step(ctx, control)
		if err != nil || !done {
			t.Fatal("enrollment dispatch", done, err)
		}
	}
	if scalar(control, "SELECT count(*) FROM edu_audit_inbox WHERE event_id=?", eventID) != "1" {
		t.Fatal("enrollment audit absent")
	}
	check(email, password, true)
	for _, fixture := range []struct {
		table, column, where, value string
		args                        []any
	}{
		{"auth_admin_enrollments", "payload", "school_id=?", "{}", []any{candidateID}},
		{"auth_admin_enrollments", "state_digest", "school_id=?", "invalid", []any{candidateID}},
		{"edu_outbox", "payload", "event_id=?", "{}", []any{eventID}},
		{"edu_audit_inbox", "tenant_id", "event_id=?", "corrupt", []any{eventID}},
		{"auth_users", "email", "school_id=?", "changed@example.test", []any{candidateID}},
		{"auth_memberships", "role", "school_id=?", "viewer", []any{candidateID}},
	} {
		original := scalar(control, "SELECT "+fixture.column+" FROM "+fixture.table+" WHERE "+fixture.where, fixture.args...)
		query := "UPDATE " + fixture.table + " SET " + fixture.column + "=? WHERE " + fixture.where
		exec(control, query, append([]any{fixture.value}, fixture.args...)...)
		check(email, password, false)
		if fixture.table == "auth_users" {
			check(fixture.value, password, false)
		}
		if got := scalar(control, "SELECT "+fixture.column+" FROM "+fixture.table+" WHERE "+fixture.where, fixture.args...); got != fixture.value {
			t.Fatal("retry restored changed state")
		}
		exec(control, query, append([]any{original}, fixture.args...)...)
	}
	originalHash := scalar(control, "SELECT password_hash FROM auth_users WHERE school_id=?", candidateID)
	changedHash, err := bcrypt.GenerateFromPassword([]byte("replacement-password-long"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	exec(control, "UPDATE auth_users SET password_hash=? WHERE school_id=?", string(changedHash), candidateID)
	check(email, password, false)
	check(email, "replacement-password-long", false)
	if scalar(control, "SELECT password_hash FROM auth_users WHERE school_id=?", candidateID) != string(changedHash) {
		t.Fatal("retry reset password")
	}
	exec(control, "UPDATE auth_users SET password_hash=? WHERE school_id=?", originalHash, candidateID)
	exec(control, "DELETE FROM auth_role_permissions WHERE school_id=? AND role='admin' AND permission='workspace.write'", candidateID)
	check(email, password, false)
	if scalar(control, "SELECT count(*) FROM auth_role_permissions WHERE school_id=? AND role='admin' AND permission='workspace.write'", candidateID) != "0" {
		t.Fatal("retry restored grant")
	}
	exec(control, "INSERT INTO auth_role_permissions (school_id,role,permission) VALUES (?,'admin','workspace.write')", candidateID)
	exec(control, "INSERT INTO ws_tenancy (school_id,id,title,status,version) VALUES (?,'unexpected','fixture','open',1)", candidateID)
	check(email, password, false)
	exec(control, "DELETE FROM ws_tenancy WHERE school_id=?", candidateID)
	// Real HTTP session creation, protected access, cross-tenant denial and logout.
	request := func(method, path, token, school string, body any, want int) []byte {
		t.Helper()
		data, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(data))
		r.Header.Set("Content-Type", "application/json")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if school != "" {
			r.Header.Set("X-School-ID", school)
		}
		w := httptest.NewRecorder()
		facades.Route().ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: got %d want %d", method, path, w.Code, want)
		}
		return w.Body.Bytes()
	}
	var session auth.Session
	if err := json.Unmarshal(request("POST", "/api/v1/session", "", "", map[string]string{"school": candidateID, "email": email, "password": password}, 200), &session); err != nil || session.Token == "" {
		t.Fatal("login", err)
	}
	request("GET", "/api/v1/modules", session.Token, "", nil, 200)
	request("GET", "/api/v1/modules/tenancy/workspace", session.Token, "", nil, 200)
	request("GET", "/api/v1/modules", session.Token, "west", nil, 403)
	check(email, password, true)
	request("DELETE", "/api/v1/session", session.Token, "", nil, 204)
	request("GET", "/api/v1/modules", session.Token, "", nil, 401)
	// Other tenants remain usable and their Auth roots do not affect retries.
	if _, err := auth.New(&control).Seed(ctx, "enrollment-neighbor", "neighbor@example.test", password); err != nil {
		t.Fatal(err)
	}
	check(email, password, true)
	if _, err := db.PinnedPool(ctx, "west"); err != nil {
		t.Fatal("unrelated placement", err)
	}
	exec(control, "DELETE FROM auth_users WHERE school_id=?", candidateID)
	check(email, password, false)
	count("auth_users", "0")
	count("auth_admin_enrollments", "1")
	t.Log("dedicated enrollment: negative state/history, atomic rollback, concurrent winner, exact retry, immutable principal/grants, control audit, HTTP access/denial/logout and tenant isolation passed")
}
