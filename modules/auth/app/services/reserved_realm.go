package services

import (
	"context"
	"database/sql"

	"github.com/elcomware/edupilot/modules/core/database"
)

// CheckEmptyRealm verifies a reserved bootstrap realm in the caller's control
// transaction. The caller must retain the provisioning lock through its write.
// Existing identities and grants require operator reconciliation.
func (s *Service) CheckEmptyRealm(ctx context.Context, tx *sql.Tx, school string) error {
	if tx == nil || !database.ValidTenant(school) {
		return ErrInvalidInput
	}
	var empty bool
	err := tx.QueryRowContext(ctx, s.pool.Bind(`SELECT
 EXISTS (SELECT 1 FROM auth_realms WHERE school_id=?)
 AND NOT EXISTS (
 SELECT 1 FROM (
 SELECT school_id FROM auth_users
 UNION ALL SELECT school_id FROM auth_roles
 UNION ALL SELECT school_id FROM auth_permissions
 UNION ALL SELECT school_id FROM auth_role_permissions
 UNION ALL SELECT school_id FROM auth_memberships
 UNION ALL SELECT school_id FROM auth_sessions
 UNION ALL SELECT school_id FROM auth_admin_enrollments
 UNION ALL SELECT realm_id FROM auth_tenant_grants
 UNION ALL SELECT tenant_id FROM auth_tenant_grants
 ) AS enrollment WHERE school_id=?)`), school, school).Scan(&empty)
	if err != nil {
		return err
	}
	if !empty {
		return ErrConflict
	}
	return nil
}

// LockEmptyRealm excludes enrollment root inserts through their realm foreign
// keys until the caller commits. Use READ COMMITTED so the emptiness query sees
// writers that committed while this row lock was being acquired.
func (s *Service) LockEmptyRealm(ctx context.Context, tx *sql.Tx, school string) error {
	if tx == nil || !database.ValidTenant(school) {
		return ErrInvalidInput
	}
	if s.pool.Driver == "postgres" {
		var locked string
		if err := tx.QueryRowContext(ctx, s.pool.Bind("SELECT school_id FROM auth_realms WHERE school_id=? FOR UPDATE"), school).Scan(&locked); err != nil {
			return err
		}
	} else {
		if _, err := tx.ExecContext(ctx, s.pool.Bind("UPDATE auth_realms SET school_id=school_id WHERE school_id=?"), school); err != nil {
			return err
		}
	}
	return s.CheckEmptyRealm(ctx, tx, school)
}
