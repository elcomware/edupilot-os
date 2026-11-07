package provisioning

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	auth "github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/outbox"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/google/uuid"
)

const activationEvent = "tenancy.tenant.activated.v1"

// DedicatedActivation describes the immutable activation commit, rather than a
// current hierarchy snapshot. CompletedAt is retained from the target event.
type DedicatedActivation struct {
	DedicatedCandidate
	OperationID      string `json:"operationId"`
	EventID          string `json:"eventId"`
	PlacementEpoch   uint64 `json:"placementEpoch"`
	HierarchyVersion uint64 `json:"hierarchyVersion"`
	CompletedAt      string `json:"completedAt"`
}

// ActivateDedicatedTenant commits the target before publishing its placement.
// The empty pin and reserved Auth realm remain fenced through both transactions.
// An exact retry can finish publication after an interrupted control commit.
func ActivateDedicatedTenant(ctx context.Context, catalog *database.Catalog, tenant scope.TenantID, expectedPool, operation string) (DedicatedActivation, error) {
	op, err := uuid.Parse(operation)
	if err != nil || op.Version() != 4 || op.Variant() != uuid.RFC4122 || op.String() != operation {
		return DedicatedActivation{}, scope.ErrInvalidID
	}
	target, err := dedicatedTarget(ctx, catalog, tenant, expectedPool)
	if err != nil {
		return DedicatedActivation{}, err
	}
	receipt := DedicatedActivation{
		DedicatedCandidate: DedicatedCandidate{Tenant: tenant, PoolID: expectedPool},
		OperationID:        operation,
		EventID:            uuid.NewSHA1(op, []byte(activationEvent)).String(),
		PlacementEpoch:     1,
		HierarchyVersion:   1,
	}
	control := catalog.Control
	// Row locks supply exclusion. READ COMMITTED observes enrollment committed
	// before the realm fence, including writers we had to wait for.
	lock, err := beginActivationTransaction(ctx, control)
	if err != nil {
		return DedicatedActivation{}, err
	}
	defer lock.Rollback()
	result, err := lock.ExecContext(ctx, control.Bind("UPDATE edu_tenant_placement SET pool_id=pool_id WHERE tenant_id=?"), tenant.String())
	if err != nil {
		return DedicatedActivation{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return DedicatedActivation{}, err
	}
	if n != 1 {
		return DedicatedActivation{}, errOccupied
	}
	var pin string
	if err := lock.QueryRowContext(ctx, control.Bind("SELECT pool_id FROM edu_tenant_placement WHERE tenant_id=?"), tenant.String()).Scan(&pin); err != nil {
		return DedicatedActivation{}, err
	}
	var stored string
	err = lock.QueryRowContext(ctx, control.Bind("SELECT payload FROM edu_tenant_activations WHERE tenant_id=? AND operation_id=? AND pool_id=?"), tenant.String(), operation, expectedPool).Scan(&stored)
	completed := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return DedicatedActivation{}, err
	}
	var reserved bool
	if err := lock.QueryRowContext(ctx, control.Bind("SELECT EXISTS (SELECT 1 FROM edu_tenant_reservations WHERE tenant_id=?)"), tenant.String()).Scan(&reserved); err != nil {
		return DedicatedActivation{}, err
	}
	if completed {
		if pin != expectedPool || reserved {
			return DedicatedActivation{}, database.ErrPlacement
		}
		receipt, err = decodeActivation(stored, receipt)
		if err != nil {
			return DedicatedActivation{}, err
		}
	} else {
		var matching bool
		if err := lock.QueryRowContext(ctx, control.Bind(`SELECT EXISTS (SELECT 1 FROM edu_tenant_reservations WHERE tenant_id=? AND operation_id=? AND pool_id=?)
   AND NOT EXISTS (SELECT 1 FROM edu_tenant_activations WHERE tenant_id=? OR operation_id=? OR pool_id=?)`), tenant.String(), operation, expectedPool, tenant.String(), operation, expectedPool).Scan(&matching); err != nil {
			return DedicatedActivation{}, err
		}
		if pin != "" || !matching {
			return DedicatedActivation{}, errOccupied
		}
	}
	if err := auth.New(&control).LockEmptyRealm(ctx, lock, tenant.String()); err != nil {
		return DedicatedActivation{}, err
	}
	if err := requireOtherPoolsEmpty(ctx, catalog, target, lock, tenant.String()); err != nil {
		return DedicatedActivation{}, err
	}
	receipt, err = activateTarget(ctx, target, receipt, completed)
	if err != nil {
		return DedicatedActivation{}, err
	}
	if !completed {
		payload, err := json.Marshal(receipt)
		if err != nil {
			return DedicatedActivation{}, err
		}
		if _, err := lock.ExecContext(ctx, control.Bind("INSERT INTO edu_tenant_activations (tenant_id,operation_id,pool_id,payload) VALUES (?,?,?,?)"), tenant.String(), operation, expectedPool, string(payload)); err != nil {
			return DedicatedActivation{}, err
		}
		if _, err := lock.ExecContext(ctx, control.Bind("UPDATE edu_tenant_placement SET pool_id=? WHERE tenant_id=?"), expectedPool, tenant.String()); err != nil {
			return DedicatedActivation{}, err
		}
		if _, err := lock.ExecContext(ctx, control.Bind("DELETE FROM edu_tenant_reservations WHERE tenant_id=? AND operation_id=? AND pool_id=?"), tenant.String(), operation, expectedPool); err != nil {
			return DedicatedActivation{}, err
		}
	}
	if err := lock.Commit(); err != nil {
		return DedicatedActivation{}, err
	}
	return receipt, nil
}

func requireOtherPoolsEmpty(ctx context.Context, catalog *database.Catalog, target database.Pool, control *sql.Tx, tenant string) error {
	for _, pool := range catalog.All() {
		if pool.DB == target.DB {
			continue
		}
		var source reader = pool.DB
		if pool.DB == catalog.Control.DB {
			source = control
		}
		if err := requireEmpty(ctx, source, pool, tenant); err != nil {
			return err
		}
	}
	return nil
}

func decodeActivation(payload string, expected DedicatedActivation) (DedicatedActivation, error) {
	var receipt DedicatedActivation
	if err := json.Unmarshal([]byte(payload), &receipt); err != nil {
		return DedicatedActivation{}, database.ErrPlacement
	}
	completedAt, err := time.Parse(time.RFC3339Nano, receipt.CompletedAt)
	if err != nil || completedAt.IsZero() || completedAt.UTC().Format(time.RFC3339Nano) != receipt.CompletedAt {
		return DedicatedActivation{}, database.ErrPlacement
	}
	expected.CompletedAt = receipt.CompletedAt
	canonical, err := json.Marshal(expected)
	if err != nil || string(canonical) != payload {
		return DedicatedActivation{}, database.ErrPlacement
	}
	return expected, nil
}

func activateTarget(ctx context.Context, pool database.Pool, receipt DedicatedActivation, completed bool) (DedicatedActivation, error) {
	tx, err := beginActivationTransaction(ctx, pool)
	if err != nil {
		return DedicatedActivation{}, err
	}
	defer tx.Rollback()
	receipt, err = reconcileActivationTarget(ctx, tx, pool, receipt, completed)
	if err != nil {
		return DedicatedActivation{}, err
	}
	if err := tx.Commit(); err != nil {
		return DedicatedActivation{}, err
	}
	return receipt, nil
}

// The caller retains the target fence through its control commit.
func reconcileActivationTarget(ctx context.Context, tx *sql.Tx, pool database.Pool, receipt DedicatedActivation, completed bool) (DedicatedActivation, error) {
	var err error
	// Freeze the installed foundation inventory, including worker audit receipts.
	// SQLite's first write acquires the database writer lock. PostgreSQL table
	// locks precede every snapshot read and exclude inserts by every owner.
	if pool.Driver == "postgres" {
		_, err = tx.ExecContext(ctx, "LOCK TABLE edu_pool_marker,tenant_gates,tenancy_organizations,tenancy_schools,tenancy_campuses,ws_tenancy,edu_outbox,edu_audit_inbox IN SHARE ROW EXCLUSIVE MODE")
	} else {
		_, err = tx.ExecContext(ctx, pool.Bind("UPDATE tenant_gates SET state=state WHERE tenant_id=?"), receipt.Tenant.String())
	}
	if err != nil {
		return DedicatedActivation{}, err
	}
	var markerMatches bool
	if err := tx.QueryRowContext(ctx, pool.Bind("SELECT EXISTS (SELECT 1 FROM edu_pool_marker WHERE singleton=1 AND owner=? AND pool_id=?)"), "tenant:"+receipt.Tenant.String(), receipt.PoolID).Scan(&markerMatches); err != nil {
		return DedicatedActivation{}, err
	}
	if !markerMatches {
		return DedicatedActivation{}, database.ErrPlacement
	}
	var poolID, state string
	var epoch, version uint64
	if err := tx.QueryRowContext(ctx, pool.Bind("SELECT pool_id,state,placement_epoch,hierarchy_version FROM tenant_gates WHERE tenant_id=?"), receipt.Tenant.String()).Scan(&poolID, &state, &epoch, &version); err != nil {
		return DedicatedActivation{}, err
	}
	if poolID != receipt.PoolID || epoch != 1 || version < 1 || (!completed && version != 1) || (state != "provisioning" && state != "active") || (completed && state != "active") {
		return DedicatedActivation{}, database.ErrPlacement
	}
	prepared, err := json.Marshal(scope.Chain{Tenant: receipt.Tenant, HierarchyVersion: 1, Active: false})
	if err != nil {
		return DedicatedActivation{}, err
	}
	if err := verifyProvisioningEvent(ctx, tx, pool, receipt.OperationID, receipt.Tenant.String(), "tenancy.tenant.prepared.v1", string(prepared)); err != nil {
		return DedicatedActivation{}, err
	}
	if state == "provisioning" {
		if err := reconcilePreparedFoundation(ctx, tx, pool, receipt.Tenant.String(), receipt.OperationID); err != nil {
			return DedicatedActivation{}, err
		}
		receipt.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
		payload, err := json.Marshal(receipt)
		if err != nil {
			return DedicatedActivation{}, err
		}
		if _, err := tx.ExecContext(ctx, pool.Bind("UPDATE tenant_gates SET state='active' WHERE tenant_id=?"), receipt.Tenant.String()); err != nil {
			return DedicatedActivation{}, err
		}
		if err := outbox.Append(ctx, tx, pool, outbox.Event{ID: receipt.EventID, TenantID: receipt.Tenant.String(), Type: activationEvent, AggregateID: receipt.Tenant.String(), Payload: payload}); err != nil {
			return DedicatedActivation{}, err
		}
	} else {
		var payload string
		if err := tx.QueryRowContext(ctx, pool.Bind("SELECT payload FROM edu_outbox WHERE event_id=? AND tenant_id=? AND aggregate_id=? AND event_type=?"), receipt.EventID, receipt.Tenant.String(), receipt.Tenant.String(), activationEvent).Scan(&payload); err != nil {
			return DedicatedActivation{}, err
		}
		verified, err := decodeActivation(payload, receipt)
		if err != nil {
			return DedicatedActivation{}, err
		}
		if completed && verified.CompletedAt != receipt.CompletedAt {
			return DedicatedActivation{}, database.ErrPlacement
		}
		receipt = verified
		if err := verifyProvisioningEvent(ctx, tx, pool, receipt.EventID, receipt.Tenant.String(), activationEvent, payload); err != nil {
			return DedicatedActivation{}, err
		}
	}
	if completed {
		var foreignOwner bool
		if err := tx.QueryRowContext(ctx, pool.Bind("SELECT EXISTS (SELECT 1 FROM "+foundationOwners+" WHERE id<>?)"), receipt.Tenant.String()).Scan(&foreignOwner); err != nil {
			return DedicatedActivation{}, err
		}
		if foreignOwner {
			return DedicatedActivation{}, errOccupied
		}
	}
	if !completed {
		var records, receipts int64
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM "+foundationOwners).Scan(&records); err != nil {
			return DedicatedActivation{}, err
		}
		if err := tx.QueryRowContext(ctx, pool.Bind(`SELECT count(*) FROM edu_audit_inbox WHERE tenant_id=? AND aggregate_id=?
   AND ((event_id=? AND event_type='tenancy.tenant.prepared.v1') OR (event_id=? AND event_type=?))`), receipt.Tenant.String(), receipt.Tenant.String(), receipt.OperationID, receipt.EventID, activationEvent).Scan(&receipts); err != nil {
			return DedicatedActivation{}, err
		}
		if records != 3+receipts {
			return DedicatedActivation{}, errOccupied
		}
	}
	return receipt, nil
}

func verifyProvisioningEvent(ctx context.Context, tx *sql.Tx, pool database.Pool, id, tenant, eventType, payload string) error {
	var stored string
	if err := tx.QueryRowContext(ctx, pool.Bind("SELECT payload FROM edu_outbox WHERE event_id=? AND tenant_id=? AND aggregate_id=? AND event_type=?"), id, tenant, tenant, eventType).Scan(&stored); err != nil {
		return err
	}
	if stored != payload {
		return database.ErrPlacement
	}
	// Historical delivery receipts may be absent, but any matching event ID or
	// provisioning type must retain the exact identity, including after edits.
	var corrupt bool
	if err := tx.QueryRowContext(ctx, pool.Bind(`SELECT EXISTS (
 SELECT 1 FROM edu_audit_inbox WHERE (event_id=? OR event_type=?)
 AND (event_id<>? OR tenant_id<>? OR aggregate_id<>? OR event_type<>?)
 UNION ALL SELECT 1 FROM edu_outbox WHERE event_type=? AND event_id<>?)`), id, eventType, id, tenant, tenant, eventType, eventType, id).Scan(&corrupt); err != nil {
		return err
	}
	if corrupt {
		return database.ErrPlacement
	}
	return nil
}

// SQLite supplies serializable writes. PostgreSQL must take a fresh snapshot
// after waiting for the explicit placement, realm and target locks.
func beginActivationTransaction(ctx context.Context, pool database.Pool) (*sql.Tx, error) {
	isolation := sql.LevelSerializable
	if pool.Driver == "postgres" {
		isolation = sql.LevelReadCommitted
	}
	return pool.DB.BeginTx(ctx, &sql.TxOptions{Isolation: isolation})
}
