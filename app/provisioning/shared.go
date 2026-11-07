// Package provisioning coordinates native operator commands across the installed foundation.
package provisioning

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/outbox"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/elcomware/edupilot/modules/tenancy/app/models"
)

var errOccupied = errors.New("tenant identifier already in use")

// A pre-existing school register or event history can outlive a lost placement
// pin. Check every configured pool before accepting a new identifier.
const foundationOwners = `(
	SELECT tenant_id AS id FROM tenant_gates
	UNION ALL SELECT tenant_id FROM tenancy_organizations
	UNION ALL SELECT tenant_id FROM tenancy_schools
	UNION ALL SELECT tenant_id FROM tenancy_campuses
	UNION ALL SELECT school_id FROM ws_tenancy
	UNION ALL SELECT tenant_id FROM edu_outbox
	UNION ALL SELECT tenant_id FROM edu_audit_inbox
) AS used`

const occupied = "SELECT EXISTS (SELECT 1 FROM " + foundationOwners + " WHERE used.id=?)"

type reader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func requireEmpty(ctx context.Context, r reader, pool database.Pool, id string) error {
	var exists bool
	if err := r.QueryRowContext(ctx, pool.Bind(occupied), id).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return errOccupied
	}
	return nil
}

// CreateSharedTenant establishes a new hierarchy on the shared/control database.
// An existing pin, including one without a gate, requires separate recovery.
func CreateSharedTenant(ctx context.Context, catalog *database.Catalog, tenant scope.TenantID, expectedPool string) (scope.Chain, error) {
	graph, err := models.NewGraph(tenant)
	if err != nil {
		return scope.Chain{}, err
	}
	chain, err := graph.Resolve(scope.Selection{Tenant: tenant})
	if err != nil {
		return scope.Chain{}, err
	}
	if err := edumigrations.CheckReady(ctx, catalog, facades.Schema().Migrations()); err != nil {
		return scope.Chain{}, err
	}
	id := tenant.String()
	if expectedPool == "" || catalog.HasDedicatedRoute(id) {
		return scope.Chain{}, database.ErrPlacement
	}
	p := catalog.Control
	for _, other := range catalog.All() {
		if other.DB != p.DB {
			if err := requireEmpty(ctx, other.DB, other, id); err != nil {
				return scope.Chain{}, err
			}
		}
	}
	tx, err := p.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return scope.Chain{}, err
	}
	defer tx.Rollback()
	// The unique pin arbitrates concurrent provisioners and existing first-use
	// callers. Bind it to the live shared marker in the insert itself.
	result, err := tx.ExecContext(ctx, p.Bind(`INSERT INTO edu_tenant_placement (tenant_id,pool_id)
		SELECT ?,pool_id FROM edu_pool_marker WHERE singleton=1 AND owner='shared' AND pool_id=?`), id, expectedPool)
	if err != nil {
		return scope.Chain{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return scope.Chain{}, err
	}
	if n != 1 {
		return scope.Chain{}, database.ErrPlacement
	}
	if err := requireEmpty(ctx, tx, p, id); err != nil {
		return scope.Chain{}, err
	}
	var realm bool
	if err := tx.QueryRowContext(ctx, p.Bind("SELECT EXISTS (SELECT 1 FROM auth_realms WHERE school_id=?)"), id).Scan(&realm); err != nil {
		return scope.Chain{}, err
	}
	if realm {
		return scope.Chain{}, errOccupied
	}
	if _, err := tx.ExecContext(ctx, p.Bind(`INSERT INTO tenant_gates (tenant_id,pool_id,placement_epoch,hierarchy_version,state)
		VALUES (?,?,1,?,'active')`), id, expectedPool, chain.HierarchyVersion); err != nil {
		return scope.Chain{}, err
	}
	payload, err := json.Marshal(chain)
	if err != nil {
		return scope.Chain{}, err
	}
	if err := outbox.Append(ctx, tx, p, outbox.Event{TenantID: id, Type: "tenancy.tenant.provisioned.v1", AggregateID: id, Payload: payload}); err != nil {
		return scope.Chain{}, err
	}
	if err := tx.Commit(); err != nil {
		return scope.Chain{}, err
	}
	return chain, nil
}
