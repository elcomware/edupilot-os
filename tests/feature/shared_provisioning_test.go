package feature

import (
	"errors"
	"testing"

	"github.com/elcomware/edupilot/app/provisioning"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/elcomware/edupilot/modules/tenancy/app/services"
)

func verifySharedProvisioning(t *testing.T, catalog *database.Catalog) {
	t.Helper()
	ctx := t.Context()
	p := catalog.Control
	tenant := requireValue(scope.NewID[scope.Tenant]())
	var marker string
	if err := p.DB.QueryRow("SELECT pool_id FROM edu_pool_marker WHERE singleton=1").Scan(&marker); err != nil {
		t.Fatal(err)
	}
	create := func(id scope.TenantID, expected string) error {
		_, err := provisioning.CreateSharedTenant(ctx, catalog, id, expected)
		return err
	}
	exec := func(pool database.Pool, q string, args ...any) {
		t.Helper()
		if _, err := pool.DB.ExecContext(ctx, pool.Bind(q), args...); err != nil {
			t.Fatal(err)
		}
	}
	assertNew := func() {
		t.Helper()
		for _, table := range []string{"edu_tenant_placement", "tenant_gates", "edu_outbox"} {
			var n int
			if err := p.DB.QueryRowContext(ctx, p.Bind("SELECT COUNT(*) FROM "+table+" WHERE tenant_id=?"), tenant.String()).Scan(&n); err != nil || n != 0 {
				t.Fatal(table, n, err)
			}
		}
	}
	if err := create(tenant, "other-pool"); !errors.Is(err, database.ErrPlacement) {
		t.Fatal("wrong physical pool", err)
	}
	assertNew()
	if _, err := catalog.ForTenantContext(ctx, tenant.String()); err != nil {
		t.Fatal(err)
	}
	if err := create(tenant, marker); err == nil {
		t.Fatal("adopted a previously pinned identifier")
	}
	var gates int
	if err := p.DB.QueryRowContext(ctx, p.Bind("SELECT count(*) FROM tenant_gates WHERE tenant_id=?"), tenant.String()).Scan(&gates); err != nil || gates != 0 {
		t.Fatal("created gate for existing pin", gates, err)
	}
	exec(p, "DELETE FROM edu_tenant_placement WHERE tenant_id=?", tenant.String())
	west := catalog.Named()["tenant_west"]
	exec(west, "INSERT INTO ws_tenancy (school_id,id,title,status,version) VALUES (?, 'synthetic', 'fixture', 'open', 1)", tenant.String())
	if err := create(tenant, marker); err == nil {
		t.Fatal("claimed records in a different pool")
	}
	assertNew()
	exec(west, "DELETE FROM ws_tenancy WHERE school_id=?", tenant.String())
	exec(p, "INSERT INTO auth_realms (school_id) VALUES (?)", tenant.String())
	if err := create(tenant, marker); err == nil {
		t.Fatal("claimed an existing school realm")
	}
	assertNew()
	exec(p, "DELETE FROM auth_realms WHERE school_id=?", tenant.String())
	exec(p, "ALTER TABLE edu_outbox RENAME COLUMN payload TO unavailable_payload")
	if err := create(tenant, marker); err == nil {
		t.Fatal("committed without a creation event")
	}
	exec(p, "ALTER TABLE edu_outbox RENAME COLUMN unavailable_payload TO payload")
	assertNew()
	results := make(chan error, 2)
	for range 2 {
		go func() { results <- create(tenant, marker) }()
	}
	first, second := <-results, <-results
	if (first == nil) == (second == nil) {
		t.Fatal("expected exactly one successful creation", first, second)
	}
	directory := services.PersistedDirectory{Pools: catalog}
	chain, err := directory.Resolve(ctx, scope.Selection{Tenant: tenant})
	if err != nil || !chain.Active || chain.Tenant != tenant || chain.HierarchyVersion != 1 {
		t.Fatal(chain, err)
	}
	if err := create(tenant, marker); err == nil {
		t.Fatal("replay overwrote existing placement")
	}
	var count int
	if err := p.DB.QueryRowContext(ctx, p.Bind("SELECT count(*) FROM edu_outbox WHERE tenant_id=? AND event_type='tenancy.tenant.provisioned.v1'"), tenant.String()).Scan(&count); err != nil || count != 1 {
		t.Fatal("unexpected creation events", count, err)
	}
}
