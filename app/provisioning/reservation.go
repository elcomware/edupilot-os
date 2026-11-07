package provisioning

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/google/uuid"
)

// DedicatedReservation records a durable pending placement. OperationID is a
// caller-generated UUID retained for recovery, never a credential.
type DedicatedReservation struct {
	DedicatedCandidate
	OperationID string `json:"operationId"`
	CreatedAt   string `json:"createdAt"`
}

// ReserveDedicatedTenant exclusively claims a fresh tenant and its auth realm.
// Repeating the same operation revalidates the target and returns the journal.
// Activation requires a separate resumable initialization protocol.
func ReserveDedicatedTenant(ctx context.Context, catalog *database.Catalog, tenant scope.TenantID, expectedPool, operation string) (DedicatedReservation, error) {
	op, err := uuid.Parse(operation)
	if err != nil || op.Version() != 4 || op.Variant() != uuid.RFC4122 || op.String() != operation {
		return DedicatedReservation{}, scope.ErrInvalidID
	}
	target, err := dedicatedTarget(ctx, catalog, tenant, expectedPool)
	if err != nil {
		return DedicatedReservation{}, err
	}
	p := catalog.Control
	tx, err := p.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return DedicatedReservation{}, err
	}
	defer tx.Rollback()
	id := tenant.String()
	// Empty is forbidden for physical markers. Existing and older resolvers
	// therefore reject this pin, including first-use callers racing this insert.
	result, err := tx.ExecContext(ctx, p.Bind(`INSERT INTO edu_tenant_placement (tenant_id,pool_id)
		VALUES (?,'') ON CONFLICT (tenant_id) DO NOTHING`), id)
	if err != nil {
		return DedicatedReservation{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return DedicatedReservation{}, err
	}
	report := DedicatedReservation{
		DedicatedCandidate: DedicatedCandidate{Tenant: tenant, PoolID: expectedPool},
		OperationID:        operation,
		CreatedAt:          time.Now().UTC().Format(time.RFC3339Nano),
	}
	if n == 0 {
		// Only the original operation may resume. An active or unrelated pin
		// cannot be turned into a reservation by supplying an operation UUID.
		err := tx.QueryRowContext(ctx, p.Bind(`SELECT r.created_at
			FROM edu_tenant_reservations r
			JOIN edu_tenant_placement p ON p.tenant_id=r.tenant_id AND p.pool_id=''
			JOIN auth_realms a ON a.school_id=r.tenant_id
			WHERE r.tenant_id=? AND r.pool_id=? AND r.operation_id=?`),
			id, expectedPool, operation).Scan(&report.CreatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return DedicatedReservation{}, errOccupied
		}
		if err != nil {
			return DedicatedReservation{}, err
		}
	} else {
		// Claim the realm in the same transaction. Seed's unique-realm claim
		// either wins first (rolling us back) or sees an empty reserved realm
		// and refuses to create an administrator.
		if _, err := tx.ExecContext(ctx, p.Bind("INSERT INTO auth_realms (school_id) VALUES (?)"), id); err != nil {
			return DedicatedReservation{}, err
		}
		if _, err := tx.ExecContext(ctx, p.Bind(`INSERT INTO edu_tenant_reservations
			(tenant_id,operation_id,pool_id,created_at) VALUES (?,?,?,?)`),
			id, operation, expectedPool, report.CreatedAt); err != nil {
			return DedicatedReservation{}, err
		}
	}
	if err := checkDedicatedData(ctx, catalog, target, id, tx); err != nil {
		return DedicatedReservation{}, err
	}
	if err := tx.Commit(); err != nil {
		return DedicatedReservation{}, err
	}
	return report, nil
}
