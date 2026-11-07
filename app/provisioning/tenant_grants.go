package provisioning

import (
	"context"
	"math"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/database/edumigrations"
	auth "github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/modules/core/database"
)

// GrantTenantHierarchy coordinates the existing placement, Auth and physical
// gate fences in that order. The target lock lasts through the control commit.
func GrantTenantHierarchy(ctx context.Context, catalog *database.Catalog, in auth.TenantGrantInput, poolID string, epoch uint64) (auth.TenantGrant, error) {
	if err := in.Validate(); err != nil {
		return auth.TenantGrant{}, err
	}
	if poolID == "" || epoch == 0 || epoch > math.MaxInt64 {
		return auth.TenantGrant{}, auth.ErrInvalidInput
	}
	if err := edumigrations.CheckReady(ctx, catalog, facades.Schema().Migrations()); err != nil {
		return auth.TenantGrant{}, err
	}
	target, err := catalog.PinnedPool(ctx, in.Tenant.String())
	if err != nil {
		return auth.TenantGrant{}, err
	}
	control := catalog.Control
	tx, err := beginActivationTransaction(ctx, control)
	if err != nil {
		return auth.TenantGrant{}, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, control.Bind(`UPDATE edu_tenant_placement SET pool_id=pool_id WHERE tenant_id=? AND pool_id=?
 AND NOT EXISTS (SELECT 1 FROM edu_tenant_reservations WHERE tenant_id=?)`), in.Tenant.String(), poolID, in.Tenant.String())
	if err != nil {
		return auth.TenantGrant{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return auth.TenantGrant{}, err
	}
	if n != 1 {
		return auth.TenantGrant{}, database.ErrPlacement
	}
	service := auth.New(&control)
	if err := service.LockTenantGrantSubject(ctx, tx, in); err != nil {
		return auth.TenantGrant{}, err
	}
	gate := tx
	if target.DB != control.DB {
		gate, err = beginActivationTransaction(ctx, target)
		if err != nil {
			return auth.TenantGrant{}, err
		}
		defer gate.Rollback()
	}
	owner := "shared"
	if target.DB != control.DB {
		owner = "tenant:" + in.Tenant.String()
	}
	result, err = gate.ExecContext(ctx, target.Bind(`UPDATE tenant_gates SET hierarchy_version=hierarchy_version
 WHERE tenant_id=? AND state='active' AND placement_epoch=? AND pool_id=?
 AND EXISTS (SELECT 1 FROM edu_pool_marker WHERE singleton=1 AND owner=? AND pool_id=?)`), in.Tenant.String(), epoch, poolID, owner, poolID)
	if err != nil {
		return auth.TenantGrant{}, err
	}
	n, err = result.RowsAffected()
	if err != nil {
		return auth.TenantGrant{}, err
	}
	if n != 1 {
		return auth.TenantGrant{}, database.ErrPlacement
	}
	receipt, err := service.CreateTenantGrant(ctx, tx, in)
	if err != nil {
		return auth.TenantGrant{}, err
	}
	if err := tx.Commit(); err != nil {
		return auth.TenantGrant{}, err
	}
	return receipt, nil
}
