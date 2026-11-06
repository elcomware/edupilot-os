package services

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

	"github.com/elcomware/edupilot/modules/auth/app/models"
	"github.com/elcomware/edupilot/modules/core/access"
	"github.com/elcomware/edupilot/modules/core/database"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid school, email or password")
	ErrInvalidInput       = errors.New("invalid school, account or password")
	ErrConflict           = errors.New("account or school already provisioned")
	ErrNotFound           = errors.New("school member not found")
)

const sessionTTL = 12 * time.Hour

type Service struct {
	pool *database.Pool
	now  func() time.Time
}

func New(pool *database.Pool) *Service { return &Service{pool: pool, now: time.Now} }

type Session struct {
	Token     string           `json:"token"`
	Principal access.Principal `json:"principal"`
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
func managedRole(role string) bool       { return role == models.RoleRegistrar || role == models.RoleViewer }

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
		return "", access.ErrUnauthenticated
	}
	b, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil || len(b) != 32 {
		return "", access.ErrUnauthenticated
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func (s *Service) Login(ctx context.Context, school, email, password string) (Session, error) {
	normalized, ok := address(email)
	if !database.ValidTenant(school) || !ok || len(password) == 0 || len(password) > 72 {
		return Session{}, ErrInvalidCredentials
	}
	const lookup = `SELECT a.id,a.password_hash,m.role FROM auth_users a
		JOIN auth_memberships m ON a.school_id=m.school_id AND a.id=m.user_id
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
	if _, err = tx.ExecContext(ctx, s.pool.Bind(`DELETE FROM auth_sessions WHERE expires_at<=?`), now.Unix()); err != nil {
		return Session{}, err
	}
	result, err := tx.ExecContext(ctx, s.pool.Bind(`INSERT INTO auth_sessions (token_hash,school_id,user_id,expires_at)
		SELECT ?,a.school_id,a.id,? FROM auth_users a JOIN auth_memberships m
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
	p, err := s.authenticate(ctx, tx, token)
	if err != nil {
		return Session{}, err
	}
	if err = tx.Commit(); err != nil {
		return Session{}, err
	}
	return Session{Token: token, Principal: p}, nil
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (s *Service) authenticate(ctx context.Context, q queryer, token string) (access.Principal, error) {
	user, err := s.currentUser(ctx, q, token)
	if err != nil {
		return access.Principal{}, err
	}
	return access.Principal{ID: user.ID, School: user.Membership.School, Permissions: user.Permissions}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (access.Principal, error) {
	return s.authenticate(ctx, s.pool.DB, token)
}

func (s *Service) Logout(ctx context.Context, token string) error {
	hashed, err := digest(token)
	if err != nil {
		return err
	}
	result, err := s.pool.DB.ExecContext(ctx, s.pool.Bind(`DELETE FROM auth_sessions WHERE token_hash=? AND expires_at>?`), hashed, s.now().Unix())
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return access.ErrUnauthenticated
	}
	return nil
}
