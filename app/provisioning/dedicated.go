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
	if tenant.IsZero() || expectedPool == "" || catalog == nil || !catalog.HasDedicatedRoute(tenant.String()) {
		return DedicatedCandidate{}, database.ErrPlacement
	}
	if err := edumigrations.CheckReady(ctx, catalog, facades.Schema().Migrations()); err != nil {
		return DedicatedCandidate{}, err
	}
	id := tenant.String()
	target := catalog.Named()["tenant_"+id]
	var matches bool
	if err := target.DB.QueryRowContext(ctx, target.Bind("SELECT EXISTS (SELECT 1 FROM edu_pool_marker WHERE singleton=1 AND owner=? AND pool_id=?)"), "tenant:"+id, expectedPool).Scan(&matches); err != nil {
		return DedicatedCandidate{}, err
	}
	if !matches {
		return DedicatedCandidate{}, database.ErrPlacement
	}
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
	for _, pool := range catalog.All() {
		if pool.DB == target.DB {
			// A dedicated destination must also be empty of other owners' data.
			if err := pool.DB.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM "+foundationOwners+")").Scan(&used); err != nil {
				return DedicatedCandidate{}, err
			}
			if used {
				return DedicatedCandidate{}, errOccupied
			}
		} else if err := requireEmpty(ctx, pool.DB, pool, id); err != nil {
			return DedicatedCandidate{}, err
		}
	}
	return DedicatedCandidate{Tenant: tenant, PoolID: expectedPool}, nil
}
