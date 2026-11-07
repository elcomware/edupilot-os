package provisioning

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/outbox"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/elcomware/edupilot/modules/tenancy/app/models"
)

// PrepareDedicatedTenant writes the initial provisioning gate and its durable
// event in the dedicated database. The control reservation stays in force.
// Retrying verifies the exact operation and target records after an uncertain
// response or process interruption. Activation is a separate protocol.
func PrepareDedicatedTenant(ctx context.Context, catalog *database.Catalog, tenant scope.TenantID, expectedPool, operation string) (scope.Chain, error) {
	target, err := dedicatedTarget(ctx, catalog, tenant, expectedPool)
	if err != nil {
		return scope.Chain{}, err
	}
	graph, err := models.NewGraph(tenant)
	if err != nil {
		return scope.Chain{}, err
	}
	chain, err := graph.Resolve(scope.Selection{Tenant: tenant})
	if err != nil {
		return scope.Chain{}, err
	}
	chain.Active = false
	payload, err := json.Marshal(chain)
	if err != nil {
		return scope.Chain{}, err
	}
	control := catalog.Control
	lock, err := control.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return scope.Chain{}, err
	}
	defer lock.Rollback()
	// The existing pin is the lock shared with competing first-use and reservation
	// operations. Keep it through target commit. Its empty value is never changed.
	result, err := lock.ExecContext(ctx, control.Bind(`UPDATE edu_tenant_placement SET pool_id=pool_id
 WHERE tenant_id=? AND pool_id='' AND EXISTS (
 SELECT 1 FROM edu_tenant_reservations r JOIN auth_realms a ON a.school_id=r.tenant_id
 WHERE r.tenant_id=? AND r.pool_id=? AND r.operation_id=?)`), tenant.String(), tenant.String(), expectedPool, operation)
	if err != nil {
		return scope.Chain{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return scope.Chain{}, err
	}
	if n != 1 {
		return scope.Chain{}, errOccupied
	}
	for _, pool := range catalog.All() {
		if pool.DB == target.DB {
			continue
		}
		var source reader = pool.DB
		if pool.DB == control.DB {
			source = lock
		}
		if err := requireEmpty(ctx, source, pool, tenant.String()); err != nil {
			return scope.Chain{}, err
		}
	}
	tx, err := target.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return scope.Chain{}, err
	}
	defer tx.Rollback()
	var poolID, state string
	var epoch, version uint64
	err = tx.QueryRowContext(ctx, target.Bind("SELECT pool_id,state,placement_epoch,hierarchy_version FROM tenant_gates WHERE tenant_id=?"), tenant.String()).Scan(&poolID, &state, &epoch, &version)
	if errors.Is(err, sql.ErrNoRows) {
		var used bool
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM "+foundationOwners+")").Scan(&used); err != nil {
			return scope.Chain{}, err
		}
		if used {
			return scope.Chain{}, errOccupied
		}
		if _, err := tx.ExecContext(ctx, target.Bind("INSERT INTO tenant_gates (tenant_id,pool_id,state,placement_epoch,hierarchy_version) VALUES (?,?,'provisioning',1,1)"), tenant.String(), expectedPool); err != nil {
			return scope.Chain{}, err
		}
		if err := outbox.Append(ctx, tx, target, outbox.Event{ID: operation, TenantID: tenant.String(), Type: "tenancy.tenant.prepared.v1", AggregateID: tenant.String(), Payload: payload}); err != nil {
			return scope.Chain{}, err
		}
	} else {
		if err != nil {
			return scope.Chain{}, err
		}
		if poolID != expectedPool || state != "provisioning" || epoch != 1 || version != chain.HierarchyVersion {
			return scope.Chain{}, database.ErrPlacement
		}
		var stored string
		if err := tx.QueryRowContext(ctx, target.Bind("SELECT payload FROM edu_outbox WHERE event_id=? AND tenant_id=? AND aggregate_id=? AND event_type='tenancy.tenant.prepared.v1'"), operation, tenant.String(), tenant.String()).Scan(&stored); err != nil {
			return scope.Chain{}, err
		}
		if stored != string(payload) {
			return scope.Chain{}, database.ErrPlacement
		}
	}
	if err := tx.Commit(); err != nil {
		return scope.Chain{}, err
	}
	return chain, nil
}
