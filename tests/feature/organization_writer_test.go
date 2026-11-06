package feature

import (
	"errors"
	"testing"

	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/elcomware/edupilot/modules/tenancy/app/models"
	"github.com/elcomware/edupilot/modules/tenancy/app/services"
)

func verifyOrganizationWriter(t *testing.T, catalog *database.Catalog) {
	t.Helper()
	ctx := t.Context()
	w := services.OrganizationWriter{Pools: catalog}
	tenant := requireValue(scope.NewID[scope.Tenant]())
	org := func(parent scope.OrganizationID) models.Organization {
		return requireValue(models.NewOrganization(requireValue(scope.NewID[scope.Organization]()), tenant, parent, "Division"))
	}
	root := org(scope.OrganizationID{})
	if _, err := w.Create(ctx, root, 1, 1); err == nil {
		t.Fatal("created an unpinned tenant")
	}
	p, err := catalog.ForTenantContext(ctx, tenant.String())
	if err != nil {
		t.Fatal(err)
	}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := p.DB.ExecContext(ctx, p.Bind(q), args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("INSERT INTO tenant_gates (tenant_id,pool_id,placement_epoch,hierarchy_version,state) SELECT ?,pool_id,3,1,'active' FROM edu_pool_marker WHERE singleton=1", tenant.String())
	if _, err := w.Create(ctx, root, 2, 1); !errors.Is(err, models.ErrConflict) {
		t.Fatal("stale epoch", err)
	}
	chain, err := w.Create(ctx, root, 3, 1)
	if err != nil || chain.HierarchyVersion != 2 || chain.Organization != root.ID() {
		t.Fatal(chain, err)
	}
	if _, err := w.Create(ctx, root, 3, 1); !errors.Is(err, models.ErrConflict) {
		t.Fatal("stale retry", err)
	}
	child := org(root.ID())
	exec("UPDATE tenant_gates SET state='maintenance' WHERE tenant_id=?", tenant.String())
	if _, err := w.Create(ctx, child, 3, 2); !errors.Is(err, models.ErrConflict) {
		t.Fatal("maintenance write", err)
	}
	exec("UPDATE tenant_gates SET state='active' WHERE tenant_id=?", tenant.String())
	exec("UPDATE tenancy_organizations SET active=false WHERE tenant_id=? AND id=?", tenant.String(), root.ID().String())
	if _, err := w.Create(ctx, child, 3, 2); !errors.Is(err, models.ErrScope) {
		t.Fatal("inactive parent write", err)
	}
	exec("UPDATE tenancy_organizations SET active=true WHERE tenant_id=? AND id=?", tenant.String(), root.ID().String())
	// Force a failure after the organization insert and version update.
	exec("ALTER TABLE edu_outbox RENAME TO drill_outbox")
	if _, err := w.Create(ctx, child, 3, 2); err == nil {
		t.Fatal("write succeeded without outbox")
	}
	exec("ALTER TABLE drill_outbox RENAME TO edu_outbox")
	// Two independent edits based on the same version may commit only once.
	done := make(chan error, 2)
	for _, candidate := range []models.Organization{child, org(root.ID())} {
		go func() { _, err := w.Create(ctx, candidate, 3, 2); done <- err }()
	}
	first, second := <-done, <-done
	if !((first == nil && errors.Is(second, models.ErrConflict)) || (second == nil && errors.Is(first, models.ErrConflict))) {
		t.Fatalf("concurrent outcomes: %v, %v", first, second)
	}
	for query, want := range map[string]int{
		"SELECT hierarchy_version FROM tenant_gates WHERE tenant_id=?":                                       3,
		"SELECT count(*) FROM tenancy_organizations WHERE tenant_id=?":                                       2,
		"SELECT count(*) FROM edu_outbox WHERE event_type='tenancy.organization.created.v1' AND tenant_id=?": 2,
	} {
		var got int
		if err := p.DB.QueryRowContext(ctx, p.Bind(query), tenant.String()).Scan(&got); err != nil || got != want {
			t.Fatal(query, got, want, err)
		}
	}
}
