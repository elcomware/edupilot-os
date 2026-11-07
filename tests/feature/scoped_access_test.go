package feature

import (
	"encoding/json"
	"errors"
	"github.com/elcomware/edupilot/app/provisioning"
	"github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/modules/core/access"
	"github.com/elcomware/edupilot/modules/core/scope"
	tenancy "github.com/elcomware/edupilot/modules/tenancy/app/services"
	"github.com/google/uuid"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/modules/core/database"
)

// Removing the real route or session fence must break the installed HTTP test.
func verifyScopedAccess(t *testing.T, catalog *database.Catalog) {
	verifyTenantGrantLifecycle(t, catalog)
	verifyScopedHTTP(t, catalog)
	verifyGrantFences(t, catalog)
	for _, path := range []string{"/api/v1/auth/contexts", "/api/v1/tenancy/hierarchy?tenant=00000000-0000-4000-8000-000000000001&kind=schools"} {
		w := httptest.NewRecorder()
		facades.Route().ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 401 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("scoped session fence %s: %d %s", path, w.Code, w.Body.String())
		}
	}
}

// Subject identity, explicit permission and durable retries must all be enforced.
func verifyTenantGrantLifecycle(t *testing.T, catalog *database.Catalog) {
	ctx := t.Context()
	control := catalog.Control
	auth := services.New(&control)
	realm := "scoped-legacy"
	password := "synthetic-password-only"
	principal := requireValue(auth.Seed(ctx, realm, "scope@example.test", password))
	session := requireValue(auth.Login(ctx, realm, "scope@example.test", password))
	tenant := requireValue(scope.NewID[scope.Tenant]())
	var pool string
	if err := control.DB.QueryRow("SELECT pool_id FROM edu_pool_marker WHERE singleton=1").Scan(&pool); err != nil {
		t.Fatal(err)
	}
	requireValue(provisioning.CreateSharedTenant(ctx, catalog, tenant, pool))
	if _, err := auth.AuthorizeTenant(ctx, session.Token, nil, tenant); !errors.Is(err, access.ErrForbidden) {
		t.Fatal("implicit admin authority", err)
	}
	input := services.TenantGrantInput{ID: uuid.NewString(), Subject: access.Subject{Realm: realm, ID: principal.ID}, Tenant: tenant, Permission: access.HierarchyRead, ExpiresAt: time.Now().Add(time.Hour).Unix()}
	receipt := requireValue(provisioning.GrantTenantHierarchy(ctx, catalog, input, pool, 1))
	retry := requireValue(provisioning.GrantTenantHierarchy(ctx, catalog, input, pool, 1))
	if receipt != retry || receipt.Version != 1 || receipt.State != "active" {
		t.Fatal("create retry", receipt, retry)
	}
	verified := requireValue(auth.AuthorizeTenant(ctx, session.Token, []string{realm}, tenant))
	if verified.Subject != input.Subject || verified.GrantID != input.ID || verified.Version != 1 {
		t.Fatal("verified subject", verified)
	}
	changed := input
	changed.ExpiresAt++
	if _, err := provisioning.GrantTenantHierarchy(ctx, catalog, changed, pool, 1); !errors.Is(err, services.ErrConflict) {
		t.Fatal("changed retry", err)
	}
	if _, err := provisioning.GrantTenantHierarchy(ctx, catalog, input, pool, 2); err == nil {
		t.Fatal("wrong epoch")
	}
	revoked := requireValue(auth.RevokeTenantGrant(ctx, input.ID, 1))
	again := requireValue(auth.RevokeTenantGrant(ctx, input.ID, 1))
	if revoked != again || revoked.Version != 2 || revoked.State != "revoked" {
		t.Fatal("revoke retry", revoked, again)
	}
	if _, err := auth.RevokeTenantGrant(ctx, input.ID, 2); !errors.Is(err, services.ErrConflict) {
		t.Fatal("invalid revoke version", err)
	}
	if _, err := auth.AuthorizeTenant(ctx, session.Token, nil, tenant); !errors.Is(err, access.ErrForbidden) {
		t.Fatal("revocation", err)
	}
	if _, err := provisioning.GrantTenantHierarchy(ctx, catalog, input, pool, 1); !errors.Is(err, services.ErrConflict) {
		t.Fatal("reactivated grant", err)
	}
	var count int
	if err := control.DB.QueryRowContext(ctx, control.Bind("SELECT count(*) FROM edu_outbox WHERE aggregate_id=?"), input.ID).Scan(&count); err != nil || count != 2 {
		t.Fatal("grant events", count, err)
	}
}

func verifyScopedHTTP(t *testing.T, catalog *database.Catalog) {
	ctx := t.Context()
	control := catalog.Control
	auth := services.New(&control)
	exec := func(p database.Pool, q string, args ...any) {
		t.Helper()
		if _, err := p.DB.ExecContext(ctx, p.Bind(q), args...); err != nil {
			t.Fatal(err)
		}
	}
	password := "synthetic-http-password"
	first := requireValue(auth.Seed(ctx, "scope-first", "same@example.test", password))
	second := requireValue(auth.Seed(ctx, "scope-second", "same@example.test", password))
	// Equal user strings and emails remain distinct composite identities.
	exec(control, "DELETE FROM auth_memberships WHERE school_id=?", "scope-second")
	exec(control, "UPDATE auth_users SET id=? WHERE school_id=? AND id=?", first.ID, "scope-second", second.ID)
	exec(control, "INSERT INTO auth_memberships (school_id,user_id,role) VALUES (?,?,'admin')", "scope-second", first.ID)
	token := requireValue(auth.Login(ctx, "scope-first", "same@example.test", password)).Token
	siblingToken := requireValue(auth.Login(ctx, "scope-second", "same@example.test", password)).Token
	get := func(path, token string, want int, headers ...string) []byte {
		t.Helper()
		r := httptest.NewRequest("GET", path, nil)
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		for i := 0; i < len(headers); i += 2 {
			r.Header.Add(headers[i], headers[i+1])
		}
		w := httptest.NewRecorder()
		facades.Route().ServeHTTP(w, r)
		if w.Code != want || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("GET %s returned %d wanted %d: %s", path, w.Code, want, w.Body.String())
		}
		return w.Body.Bytes()
	}
	shared := requireValue(scope.NewID[scope.Tenant]())
	var sharedPool string
	if err := control.DB.QueryRow("SELECT pool_id FROM edu_pool_marker WHERE singleton=1").Scan(&sharedPool); err != nil {
		t.Fatal(err)
	}
	requireValue(provisioning.CreateSharedTenant(ctx, catalog, shared, sharedPool))
	dedicated := requireValue(scope.Parse[scope.Tenant](hierarchyTenantID))
	target := requireValue(catalog.PinnedPool(ctx, dedicated.String()))
	var dedicatedPool string
	if err := target.DB.QueryRow("SELECT pool_id FROM edu_pool_marker WHERE singleton=1").Scan(&dedicatedPool); err != nil {
		t.Fatal(err)
	}
	// The metadata fixture left the dedicated hierarchy active and populated.
	root := requireValue(scope.NewID[scope.Organization]())
	exec(control, "INSERT INTO tenancy_organizations (tenant_id,id,name,active) VALUES (?,?,'Group',true)", shared.String(), root.String())
	for _, code := range []string{"A", "B"} {
		exec(control, "INSERT INTO tenancy_schools (tenant_id,id,organization_id,code,name,active) VALUES (?,?,?,?,'School',true)", shared.String(), uuid.NewString(), root.String(), code)
	}
	exec(control, "UPDATE tenant_gates SET hierarchy_version=4 WHERE tenant_id=?", shared.String())
	path := "/api/v1/tenancy/hierarchy?tenant=" + shared.String() + "&kind=schools&limit=1"
	get(path, token, 403)
	get(path, siblingToken, 403)
	// A UUID-shaped administrator realm still receives no implicit authority.
	realmAdmin := requireValue(auth.Seed(ctx, shared.String(), "admin@example.test", password))
	_ = realmAdmin
	realmToken := requireValue(auth.Login(ctx, shared.String(), "admin@example.test", password)).Token
	get(path, realmToken, 403)
	firstGrant := services.TenantGrantInput{ID: uuid.NewString(), Subject: access.Subject{Realm: "scope-first", ID: first.ID}, Tenant: shared, Permission: access.HierarchyRead, ExpiresAt: time.Now().Add(time.Hour).Unix()}
	requireValue(provisioning.GrantTenantHierarchy(ctx, catalog, firstGrant, sharedPool, 1))
	dedicatedGrant := firstGrant
	dedicatedGrant.ID = uuid.NewString()
	dedicatedGrant.Tenant = dedicated
	requireValue(provisioning.GrantTenantHierarchy(ctx, catalog, dedicatedGrant, dedicatedPool, 3))
	// A second subject has only the shared assignment.
	siblingGrant := firstGrant
	siblingGrant.ID = uuid.NewString()
	siblingGrant.Subject.Realm = "scope-second"
	requireValue(provisioning.GrantTenantHierarchy(ctx, catalog, siblingGrant, sharedPool, 1))
	get("/api/v1/tenancy/hierarchy?tenant="+dedicated.String()+"&kind=organizations", siblingToken, 403)
	var page tenancy.HierarchyPage
	if err := json.Unmarshal(get(path, token, 200), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Entries) != 1 || page.NextCursor == "" || page.PlacementEpoch != 1 || page.HierarchyVersion != 4 {
		t.Fatal(page)
	}
	cursor := page.NextCursor
	page = tenancy.HierarchyPage{}
	if err := json.Unmarshal(get(path+"&cursor="+cursor, token, 200), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Entries) != 1 || page.NextCursor != "" {
		t.Fatal("second school", page)
	}
	get(path+"&organization="+root.String(), token, 200)
	get(path+"&organization="+uuid.NewString(), token, 404)
	get("/api/v1/tenancy/hierarchy?tenant="+dedicated.String()+"&kind=schools&organization="+root.String(), token, 404)
	get("/api/v1/tenancy/hierarchy?tenant="+dedicated.String()+"&kind=schools&cursor="+cursor, token, 422)
	get(path+"&cursor=invalid", token, 422)
	get(path+"&tenant="+shared.String(), token, 422)
	get(path+"&limit=2", token, 422)
	get(path, token, 422, "X-School-ID", "scope-first", "X-School-ID", "scope-first")
	get(path, token, 422, "X-Tenant-ID", shared.String())
	get(path, token, 403, "X-School-ID", shared.String())
	get(path, token, 200, "X-School-ID", "scope-first")
	get(path+"&cursor="+strings.Repeat("A", 4096), token, 422)
	var contexts services.TenantContexts
	if err := json.Unmarshal(get("/api/v1/auth/contexts?limit=1", token, 200), &contexts); err != nil {
		t.Fatal(err)
	}
	if len(contexts.Entries) != 1 || contexts.NextCursor == "" {
		t.Fatal(contexts)
	}
	next := contexts.NextCursor
	contexts = services.TenantContexts{}
	if err := json.Unmarshal(get("/api/v1/auth/contexts?limit=1&cursor="+next, token, 200), &contexts); err != nil {
		t.Fatal(err)
	}
	if len(contexts.Entries) != 1 || contexts.NextCursor != "" {
		t.Fatal(contexts)
	}
	get("/api/v1/auth/contexts?cursor="+next, siblingToken, 422)
	get("/api/v1/auth/contexts?limit=1&limit=1", token, 422)
	get("/api/v1/auth/contexts?limit=101", token, 422)
	exec(control, "UPDATE tenant_gates SET state='maintenance' WHERE tenant_id=?", shared.String())
	get(path, token, 503)
	get("/api/v1/auth/contexts", token, 503)
	exec(control, "UPDATE tenant_gates SET state='active',hierarchy_version=5 WHERE tenant_id=?", shared.String())
	get(path+"&cursor="+cursor, token, 409)
	expiredInput := firstGrant
	expiredInput.ExpiresAt = time.Now().Unix()
	exec(control, "UPDATE auth_tenant_grants SET expires_at=? WHERE id=?", expiredInput.ExpiresAt, firstGrant.ID)
	if _, err := provisioning.GrantTenantHierarchy(ctx, catalog, expiredInput, sharedPool, 1); !errors.Is(err, services.ErrConflict) {
		t.Fatal("expired exact create retry", err)
	}
	get(path, token, 403)
	exec(control, "UPDATE auth_tenant_grants SET expires_at=? WHERE id=?", firstGrant.ExpiresAt, firstGrant.ID)
	// Snapshot-authorized in-flight read may finish after revocation. Entry after
	// the committed revocation, including a continuation, must fail.
	verifyPausedHierarchyHTTP(t, catalog, auth, token, firstGrant)
	get(path, token, 403)
	get(path+"&cursor="+cursor, token, 403)
	// A grant never leaks into legacy workspace permissions.
	exec(control, "DELETE FROM auth_role_permissions WHERE school_id=? AND permission IN ('workspace.read','workspace.write')", "scope-second")
	get("/api/v1/modules/tenancy/workspace", siblingToken, 403)
	get(path, siblingToken, 200)
	exec(control, "DELETE FROM auth_memberships WHERE school_id=? AND user_id=?", "scope-second", first.ID)
	get(path, siblingToken, 401)
	if _, err := provisioning.GrantTenantHierarchy(ctx, catalog, siblingGrant, sharedPool, 1); err == nil {
		t.Fatal("retry without membership")
	}
	exec(control, "UPDATE auth_sessions SET expires_at=? WHERE school_id=?", time.Now().Unix(), "scope-first")
	get("/api/v1/auth/contexts", token, 401)
	get("/api/v1/tenancy/hierarchy?tenant="+dedicated.String()+"&kind=schools", token, 401)
	exec(control, "UPDATE auth_sessions SET expires_at=? WHERE school_id=?", time.Now().Add(time.Hour).Unix(), "scope-first")
	if err := auth.Logout(ctx, token); err != nil {
		t.Fatal(err)
	}
	get("/api/v1/auth/contexts", token, 401)
	get("/api/v1/tenancy/hierarchy?tenant="+dedicated.String()+"&kind=organizations", token, 401)
}

func verifyGrantFences(t *testing.T, catalog *database.Catalog) {
	ctx := t.Context()
	control := catalog.Control
	auth := services.New(&control)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := control.DB.ExecContext(ctx, control.Bind(q), args...); err != nil {
			t.Fatal(err)
		}
	}
	subject := requireValue(auth.Seed(ctx, "fence-subject", "fence@example.test", "synthetic-fence-password"))
	var pool string
	if err := control.DB.QueryRow("SELECT pool_id FROM edu_pool_marker WHERE singleton=1").Scan(&pool); err != nil {
		t.Fatal(err)
	}
	tenant := requireValue(scope.NewID[scope.Tenant]())
	requireValue(provisioning.CreateSharedTenant(ctx, catalog, tenant, pool))
	in := services.TenantGrantInput{ID: uuid.NewString(), Subject: access.Subject{Realm: subject.School, ID: subject.ID}, Tenant: tenant, Permission: access.HierarchyRead, ExpiresAt: time.Now().Add(time.Hour).Unix()}
	// Pin/gate/physical failures precede any authority write.
	for _, change := range []struct{ table, column, value string }{
		{"tenant_gates", "state", "maintenance"}, {"tenant_gates", "pool_id", "wrong"}, {"edu_pool_marker", "owner", "tenant:wrong"}, {"edu_pool_marker", "pool_id", "wrong"},
	} {
		where := "tenant_id='" + tenant.String() + "'"
		if change.table == "edu_pool_marker" {
			where = "singleton=1"
		}
		var original string
		if err := control.DB.QueryRow("SELECT " + change.column + " FROM " + change.table + " WHERE " + where).Scan(&original); err != nil {
			t.Fatal(err)
		}
		exec("UPDATE "+change.table+" SET "+change.column+"=? WHERE "+where, change.value)
		_, err := provisioning.GrantTenantHierarchy(ctx, catalog, in, pool, 1)
		exec("UPDATE "+change.table+" SET "+change.column+"=? WHERE "+where, original)
		if !errors.Is(err, database.ErrPlacement) {
			t.Fatal("physical failure", change, err)
		}
	}
	exec("INSERT INTO edu_tenant_reservations (tenant_id,operation_id,pool_id,created_at) VALUES (?,?,?,'2026-11-07T00:00:00Z')", tenant.String(), uuid.NewString(), pool)
	if _, err := provisioning.GrantTenantHierarchy(ctx, catalog, in, pool, 1); err == nil {
		t.Fatal("pending reservation grant")
	}
	exec("DELETE FROM edu_tenant_reservations WHERE tenant_id=?", tenant.String())
	missing := in
	missing.ID = uuid.NewString()
	missing.Tenant = requireValue(scope.NewID[scope.Tenant]())
	if _, err := provisioning.GrantTenantHierarchy(ctx, catalog, missing, pool, 1); err == nil {
		t.Fatal("missing pin granted")
	}
	var missingPins int
	if err := control.DB.QueryRowContext(ctx, control.Bind("SELECT count(*) FROM edu_tenant_placement WHERE tenant_id=?"), missing.Tenant.String()).Scan(&missingPins); err != nil || missingPins != 0 {
		t.Fatal("grant created placement", missingPins, err)
	}
	failInsert(t, control, "edu_outbox", true)
	if _, err := provisioning.GrantTenantHierarchy(ctx, catalog, in, pool, 1); err == nil {
		t.Fatal("expected create event rollback")
	}
	failInsert(t, control, "edu_outbox", false)
	var count int
	if err := control.DB.QueryRow("SELECT count(*) FROM auth_tenant_grants WHERE id='" + in.ID + "'").Scan(&count); err != nil || count != 0 {
		t.Fatal("partial grant", count, err)
	}
	done := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { _, err := provisioning.GrantTenantHierarchy(ctx, catalog, in, pool, 1); done <- err }()
	}
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatal("concurrent exact create", err)
		}
	}
	failInsert(t, control, "edu_outbox", true)
	if _, err := auth.RevokeTenantGrant(ctx, in.ID, 1); err == nil {
		t.Fatal("expected revoke event rollback")
	}
	failInsert(t, control, "edu_outbox", false)
	var state string
	if err := control.DB.QueryRow(control.Bind("SELECT state FROM auth_tenant_grants WHERE id=?"), in.ID).Scan(&state); err != nil || state != "active" {
		t.Fatal("partial revoke", state, err)
	}
	for i := 0; i < 2; i++ {
		go func() { _, err := auth.RevokeTenantGrant(ctx, in.ID, 1); done <- err }()
	}
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatal("concurrent exact revoke", err)
		}
	}
	if err := control.DB.QueryRow(control.Bind("SELECT count(*) FROM edu_outbox WHERE aggregate_id=?"), in.ID).Scan(&count); err != nil || count != 2 {
		t.Fatal("duplicate events", count, err)
	}
	// An empty reserved realm is occupied by a grant targeting that tenant even
	// when its subject belongs to a different realm. This probes the ownership
	// inventory with deliberately injected, structurally valid control data.
	empty := requireValue(scope.NewID[scope.Tenant]())
	exec("INSERT INTO auth_realms (school_id) VALUES (?)", empty.String())
	exec("INSERT INTO edu_tenant_placement (tenant_id,pool_id) VALUES (?,'')", empty.String())
	invalid := in
	invalid.ID = uuid.NewString()
	invalid.Tenant = empty
	if _, err := provisioning.GrantTenantHierarchy(ctx, catalog, invalid, pool, 1); err == nil {
		t.Fatal("uninitialized placement granted")
	}
	exec("INSERT INTO auth_tenant_grants (id,realm_id,user_id,tenant_id,permission,state,version,expires_at,created_at) VALUES (?,?,?,? ,?,'active',1,?,?)", invalid.ID, invalid.Subject.Realm, invalid.Subject.ID, empty.String(), access.HierarchyRead, invalid.ExpiresAt, time.Now().Unix())
	tx := requireValue(control.DB.BeginTx(ctx, nil))
	err := auth.CheckEmptyRealm(ctx, tx, empty.String())
	tx.Rollback()
	if !errors.Is(err, services.ErrConflict) {
		t.Fatal("target grant bypassed empty realm fence", err)
	}
	exec("DELETE FROM auth_tenant_grants WHERE id=?", invalid.ID)
	exec("DELETE FROM edu_tenant_placement WHERE tenant_id=?", empty.String())
	exec("DELETE FROM auth_realms WHERE school_id=?", empty.String())
	t.Run("dedicated_grant_gate", func(t *testing.T) { verifyGrantGateFence(t, catalog, in.Subject) })
	for _, bad := range []services.TenantGrantInput{
		{ID: "invalid", Subject: in.Subject, Tenant: tenant, Permission: access.HierarchyRead, ExpiresAt: in.ExpiresAt},
		{ID: uuid.NewString(), Subject: in.Subject, Tenant: tenant, Permission: "workspace.read", ExpiresAt: in.ExpiresAt},
		{ID: uuid.NewString(), Subject: in.Subject, Tenant: tenant, Permission: access.HierarchyRead, ExpiresAt: -1},
	} {
		if _, err := provisioning.GrantTenantHierarchy(ctx, catalog, bad, pool, 1); err == nil {
			t.Fatal("invalid grant accepted")
		}
	}
}
