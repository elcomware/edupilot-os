package services

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"

	"github.com/elcomware/edupilot/modules/auth/app/models"
	"github.com/elcomware/edupilot/modules/core/access"
	"github.com/elcomware/edupilot/modules/core/database"
	"golang.org/x/crypto/bcrypt"
)

// Seed is a bootstrap operation for a fresh school, accessible only from a
// privileged CLI. A repeated invocation verifies the same admin and password;
// it cannot add another admin, reset a password or promote a member.
func (s *Service) Seed(ctx context.Context, school, email, password string) (access.Principal, error) {
	normalized, ok := address(email)
	if !database.ValidTenant(school) || !ok || !validNewPassword(password) {
		return access.Principal{}, ErrInvalidInput
	}
	tx, err := s.pool.DB.BeginTx(ctx, nil)
	if err != nil {
		return access.Principal{}, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, s.pool.Bind(`INSERT INTO auth_realms (school_id) VALUES (?) ON CONFLICT (school_id) DO NOTHING`), school)
	if err != nil {
		return access.Principal{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return access.Principal{}, err
	}
	if n == 0 {
		var id, hash string
		err = tx.QueryRowContext(ctx, s.pool.Bind(`SELECT a.id,a.password_hash FROM auth_users a
			JOIN auth_memberships m ON a.school_id=m.school_id AND a.id=m.user_id
			WHERE a.school_id=? AND a.email=? AND m.role=?`), school, normalized, models.RoleAdmin).Scan(&id, &hash)
		if errors.Is(err, sql.ErrNoRows) {
			return access.Principal{}, ErrConflict
		}
		if err != nil {
			return access.Principal{}, err
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
			return access.Principal{}, ErrConflict
		}
		return s.memberPrincipal(ctx, tx, school, id, models.RoleAdmin)
	}
	if err := s.seedRoles(ctx, tx, school); err != nil {
		return access.Principal{}, err
	}
	p, err := s.create(ctx, tx, school, normalized, password, models.RoleAdmin)
	if err != nil {
		return access.Principal{}, err
	}
	if err = tx.Commit(); err != nil {
		return access.Principal{}, err
	}
	return p, nil
}

func (s *Service) create(ctx context.Context, tx *sql.Tx, school, email, password, role string) (access.Principal, error) {
	b, err := randomBytes(16)
	if err != nil {
		return access.Principal{}, err
	}
	id := hex.EncodeToString(b)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return access.Principal{}, err
	}
	res, err := tx.ExecContext(ctx, s.pool.Bind(`INSERT INTO auth_users (school_id,id,email,password_hash)
		VALUES (?,?,?,?) ON CONFLICT (school_id,email) DO NOTHING`), school, id, email, string(hash))
	if err != nil {
		return access.Principal{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return access.Principal{}, err
	}
	if n != 1 {
		return access.Principal{}, ErrConflict
	}
	if _, err = tx.ExecContext(ctx, s.pool.Bind(`INSERT INTO auth_memberships (school_id,user_id,role) VALUES (?,?,?)`), school, id, role); err != nil {
		return access.Principal{}, err
	}
	return s.memberPrincipal(ctx, tx, school, id, role)
}

func (s *Service) authorizeGrant(ctx context.Context, tx *sql.Tx, token, school, role string) error {
	p, err := s.authenticate(ctx, tx, token)
	if err != nil {
		return err
	}
	if err = p.Require(school, access.ManageMembers); err != nil {
		return err
	}
	// Admin accounts are never created or modified through role management.
	if !managedRole(role) {
		return access.ErrForbidden
	}
	return nil
}

func (s *Service) CreateMember(ctx context.Context, adminToken, school, email, password, role string) (access.Principal, error) {
	tx, err := s.pool.DB.BeginTx(ctx, nil)
	if err != nil {
		return access.Principal{}, err
	}
	defer tx.Rollback()
	if err = s.authorizeGrant(ctx, tx, adminToken, school, role); err != nil {
		return access.Principal{}, err
	}
	normalized, ok := address(email)
	if !ok || !validNewPassword(password) {
		return access.Principal{}, ErrInvalidInput
	}
	p, err := s.create(ctx, tx, school, normalized, password, role)
	if err != nil {
		return access.Principal{}, err
	}
	if err = tx.Commit(); err != nil {
		return access.Principal{}, err
	}
	return p, nil
}

// SetMemberRole changes current permissions for all sessions of this member.
func (s *Service) SetMemberRole(ctx context.Context, adminToken, school, memberID, role string) error {
	tx, err := s.pool.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = s.authorizeGrant(ctx, tx, adminToken, school, role); err != nil {
		return err
	}
	var current string
	err = tx.QueryRowContext(ctx, s.pool.Bind(`SELECT role FROM auth_memberships WHERE school_id=? AND user_id=?`), school, memberID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !managedRole(current) {
		return access.ErrForbidden
	}
	res, err := tx.ExecContext(ctx, s.pool.Bind(`UPDATE auth_memberships SET role=? WHERE school_id=? AND user_id=? AND role IN ('viewer','registrar')`), role, school, memberID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return access.ErrForbidden
	}
	return tx.Commit()
}

func (s *Service) seedRoles(ctx context.Context, tx *sql.Tx, school string) error {
	for _, name := range []string{access.ModulesRead, access.WorkspaceRead, access.WorkspaceWrite, access.ManageMembers} {
		if _, err := tx.ExecContext(ctx, s.pool.Bind("INSERT INTO auth_permissions (school_id,name) VALUES (?,?)"), school, name); err != nil {
			return err
		}
	}
	for _, role := range []models.Role{models.RoleAdmin, models.RoleRegistrar, models.RoleViewer} {
		if _, err := tx.ExecContext(ctx, s.pool.Bind("INSERT INTO auth_roles (school_id,name) VALUES (?,?)"), school, role); err != nil {
			return err
		}
		grants, err := role.DefaultPermissions()
		if err != nil {
			return err
		}
		for _, name := range grants {
			if _, err := tx.ExecContext(ctx, s.pool.Bind("INSERT INTO auth_role_permissions (school_id,role,permission) VALUES (?,?,?)"), school, role, name); err != nil {
				return err
			}
		}
	}
	return nil
}
func (s *Service) memberPrincipal(ctx context.Context, tx *sql.Tx, school, id, role string) (access.Principal, error) {
	rows, err := tx.QueryContext(ctx, s.pool.Bind("SELECT permission FROM auth_role_permissions WHERE school_id=? AND role=? ORDER BY permission"), school, role)
	if err != nil {
		return access.Principal{}, err
	}
	defer rows.Close()
	p := access.Principal{ID: id, School: school, Permissions: []string{}}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return access.Principal{}, err
		}
		p.Permissions = append(p.Permissions, name)
	}
	return p, rows.Err()
}
