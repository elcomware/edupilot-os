package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"sync"
	"time"

	"github.com/elcomware/edupilot-os/platform/database"
	"golang.org/x/crypto/bcrypt"
)

const sessionTTL = 12 * time.Hour

type Service struct {
	pool *database.Pool
	now  func() time.Time
}

func New(pool *database.Pool) *Service { return &Service{pool: pool, now: time.Now} }

type Session struct {
	Token     string    `json:"token"`
	Principal Principal `json:"principal"`
}

func address(input string) (string, bool) {
	s := strings.ToLower(strings.TrimSpace(input))
	if len(s) > 254 {
		return "", false
	}
	a, err := mail.ParseAddress(s)
	return s, err == nil && a.Address == s
}

func validNewPassword(value string) bool { return len(value) >= 12 && len(value) <= 72 }
func managedRole(role string) bool       { return role == RoleRegistrar || role == RoleViewer }

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

// An absent account does the same password work as an existing account. The
// process-local dummy hash is random, uses the real work factor and cannot log in.
var dummyHash = sync.OnceValues(func() ([]byte, error) {
	b, err := randomBytes(32)
	if err != nil {
		return nil, err
	}
	return bcrypt.GenerateFromPassword(b, bcrypt.DefaultCost)
})

func digest(token string) (string, error) {
	if len(token) != 43 {
		return "", ErrUnauthenticated
	}
	b, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil || len(b) != 32 {
		return "", ErrUnauthenticated
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func (s *Service) Login(ctx context.Context, school, email, password string) (Session, error) {
	normalized, ok := address(email)
	if !database.ValidTenant(school) || !ok || len(password) == 0 || len(password) > 72 {
		return Session{}, ErrInvalidCredentials
	}
	const lookup = `SELECT a.id,a.password_hash,m.role FROM identity_accounts a
		JOIN identity_memberships m ON a.school_id=m.school_id AND a.id=m.user_id
		WHERE a.school_id=? AND a.email=?`
	var id, hash, role string
	err := s.pool.DB.QueryRowContext(ctx, s.pool.Bind(lookup), school, normalized).Scan(&id, &hash, &role)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		dummy, e := dummyHash()
		if e != nil {
			return Session{}, e
		}
		_ = bcrypt.CompareHashAndPassword(dummy, []byte(password))
		return Session{}, ErrInvalidCredentials
	case err != nil:
		return Session{}, fmt.Errorf("load membership: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return Session{}, ErrInvalidCredentials
	}
	p, err := principal(id, school, role)
	if err != nil {
		return Session{}, ErrInvalidCredentials
	}
	b, err := randomBytes(32)
	if err != nil {
		return Session{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	hashed, err := digest(token)
	if err != nil {
		return Session{}, err
	}
	// Recheck membership and password hash at issuance, after the expensive hash
	// comparison. Later requests always read current permissions from membership.
	now := s.now()
	tx, err := s.pool.DB.BeginTx(ctx, nil)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, s.pool.Bind(`DELETE FROM identity_sessions WHERE expires_at<=?`), now.Unix()); err != nil {
		return Session{}, err
	}
	result, err := tx.ExecContext(ctx, s.pool.Bind(`INSERT INTO identity_sessions (token_hash,school_id,user_id,expires_at)
		SELECT ?,a.school_id,a.id,? FROM identity_accounts a JOIN identity_memberships m
		ON a.school_id=m.school_id AND a.id=m.user_id
		WHERE a.school_id=? AND a.id=? AND a.password_hash=? AND m.role=?`),
		hashed, now.Add(sessionTTL).Unix(), school, id, hash, role)
	if err != nil {
		return Session{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return Session{}, err
	}
	if n != 1 {
		return Session{}, ErrInvalidCredentials
	}
	if err = tx.Commit(); err != nil {
		return Session{}, err
	}
	return Session{Token: token, Principal: p}, nil
}

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *Service) authenticate(ctx context.Context, q queryer, token string) (Principal, error) {
	hashed, err := digest(token)
	if err != nil {
		return Principal{}, err
	}
	const lookup = `SELECT m.user_id,m.school_id,m.role FROM identity_sessions s
		JOIN identity_memberships m ON s.school_id=m.school_id AND s.user_id=m.user_id
		WHERE s.token_hash=? AND s.expires_at>?`
	var id, school, role string
	err = q.QueryRowContext(ctx, s.pool.Bind(lookup), hashed, s.now().Unix()).Scan(&id, &school, &role)
	if errors.Is(err, sql.ErrNoRows) {
		return Principal{}, ErrUnauthenticated
	}
	if err != nil {
		return Principal{}, fmt.Errorf("load session: %w", err)
	}
	return principal(id, school, role)
}

func (s *Service) Authenticate(ctx context.Context, token string) (Principal, error) {
	return s.authenticate(ctx, s.pool.DB, token)
}

func (s *Service) Logout(ctx context.Context, token string) error {
	hashed, err := digest(token)
	if err != nil {
		return err
	}
	result, err := s.pool.DB.ExecContext(ctx, s.pool.Bind(`DELETE FROM identity_sessions WHERE token_hash=? AND expires_at>?`), hashed, s.now().Unix())
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrUnauthenticated
	}
	return nil
}

// Seed is a bootstrap operation for a fresh school, accessible only from a
// privileged CLI. A repeated invocation verifies the same admin and password;
// it cannot add another admin, reset a password or promote a member.
func (s *Service) Seed(ctx context.Context, school, email, password string) (Principal, error) {
	normalized, ok := address(email)
	if !database.ValidTenant(school) || !ok || !validNewPassword(password) {
		return Principal{}, ErrInvalidInput
	}
	tx, err := s.pool.DB.BeginTx(ctx, nil)
	if err != nil {
		return Principal{}, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, s.pool.Bind(`INSERT INTO identity_schools (school_id) VALUES (?) ON CONFLICT (school_id) DO NOTHING`), school)
	if err != nil {
		return Principal{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Principal{}, err
	}
	if n == 0 {
		var id, hash string
		err = tx.QueryRowContext(ctx, s.pool.Bind(`SELECT a.id,a.password_hash FROM identity_accounts a
			JOIN identity_memberships m ON a.school_id=m.school_id AND a.id=m.user_id
			WHERE a.school_id=? AND a.email=? AND m.role=?`), school, normalized, RoleAdmin).Scan(&id, &hash)
		if errors.Is(err, sql.ErrNoRows) {
			return Principal{}, ErrConflict
		}
		if err != nil {
			return Principal{}, err
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
			return Principal{}, ErrConflict
		}
		return principal(id, school, RoleAdmin)
	}
	p, err := s.create(ctx, tx, school, normalized, password, RoleAdmin)
	if err != nil {
		return Principal{}, err
	}
	if err = tx.Commit(); err != nil {
		return Principal{}, err
	}
	return p, nil
}

func (s *Service) create(ctx context.Context, tx *sql.Tx, school, email, password, role string) (Principal, error) {
	b, err := randomBytes(16)
	if err != nil {
		return Principal{}, err
	}
	id := hex.EncodeToString(b)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Principal{}, err
	}
	res, err := tx.ExecContext(ctx, s.pool.Bind(`INSERT INTO identity_accounts (school_id,id,email,password_hash)
		VALUES (?,?,?,?) ON CONFLICT (school_id,email) DO NOTHING`), school, id, email, string(hash))
	if err != nil {
		return Principal{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Principal{}, err
	}
	if n != 1 {
		return Principal{}, ErrConflict
	}
	if _, err = tx.ExecContext(ctx, s.pool.Bind(`INSERT INTO identity_memberships (school_id,user_id,role) VALUES (?,?,?)`), school, id, role); err != nil {
		return Principal{}, err
	}
	return principal(id, school, role)
}

func (s *Service) authorizeGrant(ctx context.Context, tx *sql.Tx, token, school, role string) error {
	p, err := s.authenticate(ctx, tx, token)
	if err != nil {
		return err
	}
	if err = p.Require(school, ManageMembers); err != nil {
		return err
	}
	// Admin accounts are never created or modified through role management.
	if !managedRole(role) {
		return ErrForbidden
	}
	return nil
}

func (s *Service) CreateMember(ctx context.Context, adminToken, school, email, password, role string) (Principal, error) {
	tx, err := s.pool.DB.BeginTx(ctx, nil)
	if err != nil {
		return Principal{}, err
	}
	defer tx.Rollback()
	if err = s.authorizeGrant(ctx, tx, adminToken, school, role); err != nil {
		return Principal{}, err
	}
	normalized, ok := address(email)
	if !ok || !validNewPassword(password) {
		return Principal{}, ErrInvalidInput
	}
	p, err := s.create(ctx, tx, school, normalized, password, role)
	if err != nil {
		return Principal{}, err
	}
	if err = tx.Commit(); err != nil {
		return Principal{}, err
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
	err = tx.QueryRowContext(ctx, s.pool.Bind(`SELECT role FROM identity_memberships WHERE school_id=? AND user_id=?`), school, memberID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !managedRole(current) {
		return ErrForbidden
	}
	res, err := tx.ExecContext(ctx, s.pool.Bind(`UPDATE identity_memberships SET role=? WHERE school_id=? AND user_id=? AND role IN ('viewer','registrar')`), role, school, memberID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrForbidden
	}
	return tx.Commit()
}
