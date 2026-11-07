package provisioning

import (
	"context"
	"encoding/json"

	auth "github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/google/uuid"
)

// EnrollDedicatedAdmin is the privileged native entry point after activation.
// Both database fences remain held until Auth, receipt and event commit in control.
func EnrollDedicatedAdmin(ctx context.Context, catalog *database.Catalog, tenant scope.TenantID, expectedPool, operation, email, password string) (auth.AdminEnrollment, error) {
	op, err := uuid.Parse(operation)
	if err != nil || op.Version() != 4 || op.Variant() != uuid.RFC4122 || op.String() != operation {
		return auth.AdminEnrollment{}, scope.ErrInvalidID
	}
	target, err := dedicatedTarget(ctx, catalog, tenant, expectedPool)
	if err != nil {
		return auth.AdminEnrollment{}, err
	}
	control := catalog.Control
	tx, err := beginActivationTransaction(ctx, control)
	if err != nil {
		return auth.AdminEnrollment{}, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, control.Bind("UPDATE edu_tenant_placement SET pool_id=pool_id WHERE tenant_id=? AND pool_id=?"), tenant.String(), expectedPool)
	if err != nil {
		return auth.AdminEnrollment{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return auth.AdminEnrollment{}, err
	}
	if n != 1 {
		return auth.AdminEnrollment{}, database.ErrPlacement
	}
	var stored string
	if err := tx.QueryRowContext(ctx, control.Bind(`SELECT payload FROM edu_tenant_activations WHERE tenant_id=? AND pool_id=? AND operation_id=? AND NOT EXISTS (SELECT 1 FROM edu_tenant_reservations WHERE tenant_id=?)`), tenant.String(), expectedPool, operation, tenant.String()).Scan(&stored); err != nil {
		return auth.AdminEnrollment{}, err
	}
	activation, err := decodeActivation(stored, DedicatedActivation{DedicatedCandidate: DedicatedCandidate{Tenant: tenant, PoolID: expectedPool}, OperationID: operation, EventID: uuid.NewSHA1(op, []byte(activationEvent)).String(), PlacementEpoch: 1, HierarchyVersion: 1})
	if err != nil {
		return auth.AdminEnrollment{}, err
	}
	service := auth.New(&control)
	if err := service.LockEnrollmentRealm(ctx, tx, tenant.String()); err != nil {
		return auth.AdminEnrollment{}, err
	}
	// Freeze control-side foundation data too. Only this enrollment's event and
	// optional matching worker receipt may live here for the dedicated tenant.
	if control.Driver == "postgres" {
		if _, err := tx.ExecContext(ctx, "LOCK TABLE tenant_gates,tenancy_organizations,tenancy_schools,tenancy_campuses,ws_tenancy,edu_outbox,edu_audit_inbox IN SHARE ROW EXCLUSIVE MODE"); err != nil {
			return auth.AdminEnrollment{}, err
		}
	}
	for _, pool := range catalog.All() {
		if pool.DB == target.DB || pool.DB == control.DB {
			continue
		}
		if err := requireEmpty(ctx, pool.DB, pool, tenant.String()); err != nil {
			return auth.AdminEnrollment{}, err
		}
	}
	targetTx, err := beginActivationTransaction(ctx, target)
	if err != nil {
		return auth.AdminEnrollment{}, err
	}
	defer targetTx.Rollback()
	if _, err := reconcileActivationTarget(ctx, targetTx, target, activation, true); err != nil {
		return auth.AdminEnrollment{}, err
	}
	receipt, err := service.EnrollInitialAdmin(ctx, tx, tenant.String(), expectedPool, operation, email, password)
	if err != nil {
		return auth.AdminEnrollment{}, err
	}
	payload, err := json.Marshal(receipt)
	if err != nil {
		return auth.AdminEnrollment{}, err
	}
	// Scope historical type checks to this tenant because Auth events from other
	// dedicated enrollments legitimately share the control pool.
	var records, allowed int
	if err := tx.QueryRowContext(ctx, control.Bind("SELECT count(*) FROM "+foundationOwners+" WHERE id=?"), tenant.String()).Scan(&records); err != nil {
		return auth.AdminEnrollment{}, err
	}
	if err := tx.QueryRowContext(ctx, control.Bind(`SELECT count(*) FROM (
 SELECT event_id FROM edu_outbox WHERE event_id=? AND tenant_id=? AND aggregate_id=? AND event_type=? AND payload=?
 UNION ALL SELECT event_id FROM edu_audit_inbox WHERE event_id=? AND tenant_id=? AND aggregate_id=? AND event_type=?) AS permitted`), receipt.EventID, tenant.String(), tenant.String(), auth.AdminEnrollmentEvent, string(payload), receipt.EventID, tenant.String(), tenant.String(), auth.AdminEnrollmentEvent).Scan(&allowed); err != nil {
		return auth.AdminEnrollment{}, err
	}
	var corrupt bool
	if err := tx.QueryRowContext(ctx, control.Bind("SELECT EXISTS (SELECT 1 FROM edu_audit_inbox WHERE event_id=? AND (tenant_id<>? OR aggregate_id<>? OR event_type<>?))"), receipt.EventID, tenant.String(), tenant.String(), auth.AdminEnrollmentEvent).Scan(&corrupt); err != nil {
		return auth.AdminEnrollment{}, err
	}
	if records != allowed || corrupt {
		return auth.AdminEnrollment{}, errOccupied
	}
	if err := tx.Commit(); err != nil {
		return auth.AdminEnrollment{}, err
	}
	return receipt, nil
}
