package provisioning

import (
	"context"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
)

// DedicatedCandidate is the observed target of a read-only preflight.
type DedicatedCandidate struct {
	Tenant scope.TenantID `json:"tenant"`
	PoolID string         `json:"poolId"`
}

// CheckDedicatedTenant inspects all configured pools without acquiring a pin.
// Provisioning must revalidate these observations when it reserves the tenant.
func CheckDedicatedTenant(ctx context.Context, catalog *database.Catalog, tenant scope.TenantID, expectedPool string) (DedicatedCandidate, error) {
	target, err := dedicatedTarget(ctx, catalog, tenant, expectedPool)
	if err != nil {
		return DedicatedCandidate{}, err
	}
	id := tenant.String()
	control := catalog.Control
	var used bool
	if err := control.DB.QueryRowContext(ctx, control.Bind(`SELECT EXISTS (
		SELECT 1 FROM edu_tenant_placement WHERE tenant_id=? OR pool_id=?
		UNION ALL SELECT 1 FROM auth_realms WHERE school_id=?)`), id, expectedPool, id).Scan(&used); err != nil {
		return DedicatedCandidate{}, err
	}
	if used {
		return DedicatedCandidate{}, errOccupied
	}
	if err := checkDedicatedData(ctx, catalog, target, id, control.DB); err != nil {
		return DedicatedCandidate{}, err
	}
	return DedicatedCandidate{Tenant: tenant, PoolID: expectedPool}, nil
}

// dedicatedTarget rechecks installed receipts and the physical owner/identity.
func dedicatedTarget(ctx context.Context, catalog *database.Catalog, tenant scope.TenantID, expectedPool string) (database.Pool, error) {
	if tenant.IsZero() || expectedPool == "" || catalog == nil || !catalog.HasDedicatedRoute(tenant.String()) {
		return database.Pool{}, database.ErrPlacement
	}
	if err := edumigrations.CheckReady(ctx, catalog, facades.Schema().Migrations()); err != nil {
		return database.Pool{}, err
	}
	id := tenant.String()
	target := catalog.Named()["tenant_"+id]
	var matches bool
	if err := target.DB.QueryRowContext(ctx, target.Bind("SELECT EXISTS (SELECT 1 FROM edu_pool_marker WHERE singleton=1 AND owner=? AND pool_id=?)"), "tenant:"+id, expectedPool).Scan(&matches); err != nil {
		return database.Pool{}, err
	}
	if !matches {
		return database.Pool{}, database.ErrPlacement
	}
	return target, nil
}

// The caller supplies its control transaction after acquiring the unique pin.
func checkDedicatedData(ctx context.Context, catalog *database.Catalog, target database.Pool, id string, control reader) error {
	var used bool
	for _, pool := range catalog.All() {
		if pool.DB == target.DB {
			// A dedicated destination must also be empty of other owners' data.
			if err := pool.DB.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM "+foundationOwners+")").Scan(&used); err != nil {
				return err
			}
			if used {
				return errOccupied
			}
		} else {
			var source reader = pool.DB
			if pool.DB == catalog.Control.DB {
				source = control
			}
			if err := requireEmpty(ctx, source, pool, id); err != nil {
				return err
			}
		}
	}
	return nil
}
