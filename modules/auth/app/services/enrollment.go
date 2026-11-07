package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/elcomware/edupilot/modules/auth/app/models"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/outbox"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const AdminEnrollmentEvent = "auth.admin.enrolled.v1"

// AdminEnrollment is a credential-free, immutable operator receipt.
type AdminEnrollment struct {
	Version     int    `json:"version"`
	Tenant      string `json:"tenant"`
	PoolID      string `json:"poolId"`
	OperationID string `json:"operationId"`
	EventID     string `json:"eventId"`
	UserID      string `json:"userId"`
	CompletedAt string `json:"completedAt"`
}

// LockEnrollmentRealm also fences updates to existing principals and grants.
// Realm FK locks alone exclude new roots but do not exclude such updates.
// The caller holds placement first and uses READ COMMITTED on PostgreSQL.
func (s *Service) LockEnrollmentRealm(ctx context.Context, tx *sql.Tx, school string) error {
	if tx == nil || !database.ValidTenant(school) {
		return ErrInvalidInput
	}
	if s.pool.Driver == "postgres" {
		if _, err := tx.ExecContext(ctx, "LOCK TABLE auth_realms,auth_users,auth_roles,auth_permissions,auth_role_permissions,auth_memberships,auth_sessions,auth_admin_enrollments,auth_tenant_grants IN SHARE ROW EXCLUSIVE MODE"); err != nil {
			return err
		}
		var locked string
		return tx.QueryRowContext(ctx, s.pool.Bind("SELECT school_id FROM auth_realms WHERE school_id=? FOR UPDATE"), school).Scan(&locked)
	}
	result, err := tx.ExecContext(ctx, s.pool.Bind("UPDATE auth_realms SET school_id=school_id WHERE school_id=?"), school)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrConflict
	}
	return nil
}

// EnrollInitialAdmin requires the caller's placement, Auth and target fences
// and a verified completed activation. It never commits the supplied transaction.
func (s *Service) EnrollInitialAdmin(ctx context.Context, tx *sql.Tx, school, pool, operation, email, password string) (AdminEnrollment, error) {
	normalized, ok := address(email)
	op, err := uuid.Parse(operation)
	if tx == nil || !database.ValidTenant(school) || pool == "" || err != nil || op.Version() != 4 || op.Variant() != uuid.RFC4122 || op.String() != operation || !ok || !validNewPassword(password) {
		return AdminEnrollment{}, ErrInvalidInput
	}
	// New tenant grants are a later authority transition, outside the immutable
	// v1 enrollment receipt, whether this realm is their subject or target.
	var progressed bool
	if err := tx.QueryRowContext(ctx, s.pool.Bind("SELECT EXISTS (SELECT 1 FROM auth_tenant_grants WHERE realm_id=? OR tenant_id=?)"), school, school).Scan(&progressed); err != nil {
		return AdminEnrollment{}, err
	}
	if progressed {
		return AdminEnrollment{}, ErrConflict
	}
	expected := AdminEnrollment{Version: 1, Tenant: school, PoolID: pool, OperationID: operation, EventID: uuid.NewSHA1(op, []byte(AdminEnrollmentEvent)).String()}
	var payload, digest string
	err = tx.QueryRowContext(ctx, s.pool.Bind("SELECT payload,state_digest FROM auth_admin_enrollments WHERE school_id=? AND operation_id=?"), school, operation).Scan(&payload, &digest)
	if err == nil {
		return s.retryEnrollment(ctx, tx, expected, payload, digest, normalized, password)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return AdminEnrollment{}, err
	}
	var existing bool
	if err := tx.QueryRowContext(ctx, s.pool.Bind("SELECT EXISTS (SELECT 1 FROM auth_admin_enrollments WHERE school_id=? OR operation_id=?)"), school, operation).Scan(&existing); err != nil {
		return AdminEnrollment{}, err
	}
	if existing {
		return AdminEnrollment{}, ErrConflict
	}
	// A missing enrollment receipt never authorizes adoption of orphan history.
	if err := tx.QueryRowContext(ctx, s.pool.Bind(`SELECT EXISTS (
 SELECT 1 FROM edu_outbox WHERE tenant_id=? OR event_id=?
 UNION ALL SELECT 1 FROM edu_audit_inbox WHERE tenant_id=? OR event_id=?)`), school, expected.EventID, school, expected.EventID).Scan(&existing); err != nil {
		return AdminEnrollment{}, err
	}
	if existing {
		return AdminEnrollment{}, ErrConflict
	}
	if err := s.CheckEmptyRealm(ctx, tx, school); err != nil {
		return AdminEnrollment{}, err
	}
	if err := s.seedRoles(ctx, tx, school); err != nil {
		return AdminEnrollment{}, err
	}
	principal, err := s.create(ctx, tx, school, normalized, password, models.RoleAdmin)
	if err != nil {
		return AdminEnrollment{}, err
	}
	expected.UserID = principal.ID
	expected.CompletedAt = s.now().UTC().Format(time.RFC3339Nano)
	state, err := s.enrollmentDigest(ctx, tx, school)
	if err != nil {
		return AdminEnrollment{}, err
	}
	encoded, err := json.Marshal(expected)
	if err != nil {
		return AdminEnrollment{}, err
	}
	if _, err := tx.ExecContext(ctx, s.pool.Bind("INSERT INTO auth_admin_enrollments (school_id,operation_id,payload,state_digest) VALUES (?,?,?,?)"), school, operation, string(encoded), state); err != nil {
		return AdminEnrollment{}, err
	}
	if err := outbox.Append(ctx, tx, *s.pool, outbox.Event{ID: expected.EventID, TenantID: school, AggregateID: school, Type: AdminEnrollmentEvent, Payload: encoded}); err != nil {
		return AdminEnrollment{}, err
	}
	return expected, nil
}

func (s *Service) retryEnrollment(ctx context.Context, tx *sql.Tx, expected AdminEnrollment, payload, digest, email, password string) (AdminEnrollment, error) {
	var stored AdminEnrollment
	if err := json.Unmarshal([]byte(payload), &stored); err != nil {
		return AdminEnrollment{}, ErrConflict
	}
	completed, err := time.Parse(time.RFC3339Nano, stored.CompletedAt)
	userID, parseErr := hex.DecodeString(stored.UserID)
	if err != nil || completed.IsZero() || completed.UTC().Format(time.RFC3339Nano) != stored.CompletedAt || parseErr != nil || len(userID) != 16 || hex.EncodeToString(userID) != stored.UserID {
		return AdminEnrollment{}, ErrConflict
	}
	expected.UserID = stored.UserID
	expected.CompletedAt = stored.CompletedAt
	canonical, err := json.Marshal(expected)
	if err != nil || string(canonical) != payload {
		return AdminEnrollment{}, ErrConflict
	}
	state, err := s.enrollmentDigest(ctx, tx, expected.Tenant)
	if err != nil {
		return AdminEnrollment{}, err
	}
	if state != digest {
		return AdminEnrollment{}, ErrConflict
	}
	var hash string
	err = tx.QueryRowContext(ctx, s.pool.Bind(`SELECT a.password_hash FROM auth_users a JOIN auth_memberships m ON a.school_id=m.school_id AND a.id=m.user_id WHERE a.school_id=? AND a.id=? AND a.email=? AND m.role=?`), expected.Tenant, expected.UserID, email, models.RoleAdmin).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return AdminEnrollment{}, ErrConflict
	}
	if err != nil {
		return AdminEnrollment{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return AdminEnrollment{}, ErrConflict
	}
	var event string
	if err := tx.QueryRowContext(ctx, s.pool.Bind("SELECT payload FROM edu_outbox WHERE event_id=? AND tenant_id=? AND aggregate_id=? AND event_type=?"), expected.EventID, expected.Tenant, expected.Tenant, AdminEnrollmentEvent).Scan(&event); err != nil {
		return AdminEnrollment{}, err
	}
	if event != payload {
		return AdminEnrollment{}, ErrConflict
	}
	return expected, nil
}

// This private versioned commitment detects any change to the initial account
// and role state. It contains no recoverable credentials or duplicate password
// verifier. Sessions are intentionally excluded so ordinary login/logout works.
func (s *Service) enrollmentDigest(ctx context.Context, tx *sql.Tx, school string) (string, error) {
	hash := sha256.New()
	encoder := json.NewEncoder(hash)
	if err := encoder.Encode("auth-enrollment-state-v1"); err != nil {
		return "", err
	}
	for _, query := range []string{
		"SELECT id,email,password_hash FROM auth_users WHERE school_id=? ORDER BY id",
		"SELECT name FROM auth_roles WHERE school_id=? ORDER BY name",
		"SELECT name FROM auth_permissions WHERE school_id=? ORDER BY name",
		"SELECT role,permission FROM auth_role_permissions WHERE school_id=? ORDER BY role,permission",
		"SELECT user_id,role FROM auth_memberships WHERE school_id=? ORDER BY user_id",
	} {
		rows, err := tx.QueryContext(ctx, s.pool.Bind(query), school)
		if err != nil {
			return "", err
		}
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			return "", err
		}
		records := [][]string{}
		for rows.Next() {
			values := make([]string, len(columns))
			dest := make([]any, len(columns))
			for i := range values {
				dest[i] = &values[i]
			}
			if err := rows.Scan(dest...); err != nil {
				rows.Close()
				return "", err
			}
			records = append(records, values)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return "", err
		}
		if err := encoder.Encode(records); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
