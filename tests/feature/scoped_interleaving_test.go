package feature

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/elcomware/edupilot/app/provisioning"
	auth "github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/modules/core/access"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/elcomware/edupilot/modules/tenancy/app/http/controllers"
	tenancy "github.com/elcomware/edupilot/modules/tenancy/app/services"
	tenancyroutes "github.com/elcomware/edupilot/modules/tenancy/routes"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// This adapter pauses after the real Auth snapshot has committed. It injects no
// production hook and runs the real Tenancy HTTP route/controller/reader.
type pausedTenantAuthorizer struct {
	service    *auth.Service
	authorized chan struct{}
	release    chan struct{}
}

func (p pausedTenantAuthorizer) AuthorizeTenant(ctx context.Context, token string, realms []string, tenant scope.TenantID) (access.TenantAuthorization, error) {
	verified, err := p.service.AuthorizeTenant(ctx, token, realms, tenant)
	if err != nil {
		return verified, err
	}
	p.authorized <- struct{}{}
	select {
	case <-p.release:
		return verified, nil
	case <-ctx.Done():
		return access.TenantAuthorization{}, ctx.Err()
	}
}

func verifyPausedHierarchyHTTP(t *testing.T, catalog *database.Catalog, service *auth.Service, token string, grant auth.TenantGrantInput) {
	t.Helper()
	paused := pausedTenantAuthorizer{service: service, authorized: make(chan struct{}, 1), release: make(chan struct{})}
	controller := controllers.HierarchyController{Authorizer: func() (access.TenantAuthorizer, error) { return paused, nil }, Reader: func() (tenancy.HierarchyReader, error) { return tenancy.HierarchyReader{Pools: catalog}, nil }}
	server := httptest.NewServer(tenancyroutes.HTTP(controller, http.NotFoundHandler()))
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	request := func() int {
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/v1/tenancy/hierarchy?tenant="+grant.Tenant.String()+"&kind=schools", nil)
		if err != nil {
			return 0
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := server.Client().Do(req)
		if err != nil {
			return 0
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode
	}
	done := make(chan int, 1)
	go func() { done <- request() }()
	select {
	case <-paused.authorized:
	case <-ctx.Done():
		t.Fatal("HTTP request never reached authorization snapshot")
	}
	if _, err := service.RevokeTenantGrant(ctx, grant.ID, 1); err != nil {
		close(paused.release)
		t.Fatal(err)
	}
	close(paused.release)
	if status := <-done; status != 200 {
		t.Fatal("authorized in-flight HTTP read", status)
	}
	if status := request(); status != 403 {
		t.Fatal("post-commit HTTP entry accepted revoked grant", status)
	}
}

// PostgreSQL's advisory-lock barrier stops the real control outbox INSERT after
// the dedicated gate has been locked, before authority can commit. A competing
// target writer must time out until that control transaction completes.
func verifyGrantGateFence(t *testing.T, catalog *database.Catalog, subject access.Subject) {
	t.Helper()
	if catalog.Control.Driver != "postgres" {
		t.Skip("dedicated gate interleaving requires PostgreSQL advisory/table locks")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	control := catalog.Control
	tenant := requireValue(scope.Parse[scope.Tenant](hierarchyTenantID))
	target := requireValue(catalog.PinnedPool(ctx, tenant.String()))
	var pool string
	var epoch uint64
	if err := target.DB.QueryRowContext(ctx, target.Bind("SELECT pool_id,placement_epoch FROM tenant_gates WHERE tenant_id=?"), tenant.String()).Scan(&pool, &epoch); err != nil {
		t.Fatal(err)
	}
	barrier := requireValue(control.DB.BeginTx(ctx, nil))
	defer barrier.Rollback()
	const key = 7912371
	if _, err := barrier.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", key); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		"CREATE FUNCTION scoped_grant_pause() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.event_type='auth.tenant-grant.created.v1' THEN PERFORM pg_advisory_xact_lock(7912371); END IF; RETURN NEW; END $$",
		"CREATE TRIGGER scoped_grant_pause BEFORE INSERT ON edu_outbox FOR EACH ROW EXECUTE FUNCTION scoped_grant_pause()",
	} {
		if _, err := control.DB.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		barrier.Rollback()
		_, _ = control.DB.ExecContext(context.Background(), "DROP TRIGGER scoped_grant_pause ON edu_outbox")
		_, _ = control.DB.ExecContext(context.Background(), "DROP FUNCTION scoped_grant_pause()")
	}()
	in := auth.TenantGrantInput{ID: uuid.NewString(), Subject: subject, Tenant: tenant, Permission: access.HierarchyRead, ExpiresAt: time.Now().Add(time.Hour).Unix()}
	done := make(chan error, 1)
	go func() { _, err := provisioning.GrantTenantHierarchy(ctx, catalog, in, pool, epoch); done <- err }()
	for {
		var waiting bool
		err := control.DB.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_locks WHERE locktype='advisory' AND objid=$1 AND NOT granted AND database=(SELECT oid FROM pg_database WHERE datname=current_database()))`, key).Scan(&waiting)
		if err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		select {
		case err := <-done:
			t.Fatal("grant finished before control barrier", err)
		case <-ctx.Done():
			t.Fatal("grant never reached control barrier", ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}
	var count int
	if err := control.DB.QueryRowContext(ctx, control.Bind("SELECT count(*) FROM auth_tenant_grants WHERE id=?"), in.ID).Scan(&count); err != nil || count != 0 {
		t.Fatal("grant visible before control commit", count, err)
	}
	competing := requireValue(target.DB.BeginTx(ctx, nil))
	if _, err := competing.ExecContext(ctx, "SET LOCAL lock_timeout='200ms'"); err != nil {
		competing.Rollback()
		t.Fatal(err)
	}
	_, err := competing.ExecContext(ctx, target.Bind("UPDATE tenant_gates SET hierarchy_version=hierarchy_version WHERE tenant_id=?"), tenant.String())
	competing.Rollback()
	var pgerr *pgconn.PgError
	if !errors.As(err, &pgerr) || pgerr.Code != "55P03" {
		t.Fatal("target gate not retained through control commit", err)
	}
	if err := barrier.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal("grant commit after release", err)
	}
	if _, err := target.DB.ExecContext(ctx, target.Bind("UPDATE tenant_gates SET hierarchy_version=hierarchy_version WHERE tenant_id=?"), tenant.String()); err != nil {
		t.Fatal("target gate did not release", err)
	}
}
