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
 ) AS enrollment WHERE school_id=?)`), school, school).Scan(&empty)
	if err != nil {
		return err
	}
	if !empty {
		return ErrConflict
	}
	return nil
}
