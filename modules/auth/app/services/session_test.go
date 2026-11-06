package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/elcomware/edupilot/modules/auth/app/models"
	"github.com/elcomware/edupilot/modules/auth/database/migrations"
	"github.com/elcomware/edupilot/modules/core/access"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/tests/schemafixture"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"golang.org/x/crypto/bcrypt"
)

func fixture(t *testing.T) *Service {
	t.Helper()
	dsn := database.SQLiteFileDSN(filepath.Join(t.TempDir(), "identity.sqlite"), url.Values{"_pragma": {"foreign_keys(ON)"}})
	db, err := driver.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if err := migrations.Accounts(schemafixture.Builder{DB: db}); err != nil {
		t.Fatal(err)
	}
	return New(&database.Pool{DB: db, Driver: "sqlite"})
}

func secret(t *testing.T) string {
	t.Helper()
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

func login(t *testing.T, s *Service, school, email, password string) Session {
	t.Helper()
	session, err := s.Login(t.Context(), school, email, password)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func TestSessionLifecycleAndStorage(t *testing.T) {
	s := fixture(t)
	ctx := t.Context()
	password := secret(t)
	now := time.Unix(1770000000, 0)
	s.now = func() time.Time { return now }
	p, err := s.Seed(ctx, "north", "admin@example.test", password)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Require("north", access.ManageMembers); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Seed(ctx, "north", "ADMIN@example.test", password); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ email, password string }{{"other@example.test", password}, {"admin@example.test", secret(t)}} {
		if _, err := s.Seed(ctx, "north", tc.email, tc.password); !errors.Is(err, ErrConflict) {
			t.Fatalf("bootstrap replaced an existing admin: %v", err)
		}
	}

	var passwordHash string
	if err := s.pool.DB.QueryRowContext(ctx, "SELECT password_hash FROM auth_users WHERE school_id=?", "north").Scan(&passwordHash); err != nil {
		t.Fatal(err)
	}
	if password == passwordHash || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		t.Fatal("password verifier was not stored")
	}
	for _, tc := range []struct{ school, email, password string }{{"south", "admin@example.test", password}, {"north", "missing@example.test", password}, {"north", "admin@example.test", secret(t)}} {
		if _, err := s.Login(ctx, tc.school, tc.email, tc.password); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("unexpected login result: %v", err)
		}
	}
	first := login(t, s, "north", " admin@EXAMPLE.test ", password)
	second := login(t, s, "north", "admin@example.test", password)
	if first.Token == second.Token {
		t.Fatal("sessions reused the same token")
	}
	hash, err := digest(first.Token)
	if err != nil {
		t.Fatal(err)
	}
	var stored string
	if err = s.pool.DB.QueryRowContext(ctx, "SELECT token_hash FROM auth_sessions WHERE token_hash=?", hash).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == first.Token {
		t.Fatal("stored reusable session token")
	}
	got, err := s.Authenticate(ctx, first.Token)
	if err != nil || got.ID != p.ID {
		t.Fatalf("authenticate mismatch: %v", err)
	}
	if err := s.Logout(ctx, first.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, first.Token); !errors.Is(err, access.ErrUnauthenticated) {
		t.Fatalf("revoked token authenticated: %v", err)
	}
	now = now.Add(sessionTTL)
	if _, err := s.Authenticate(ctx, second.Token); !errors.Is(err, access.ErrUnauthenticated) {
		t.Fatalf("expired token authenticated: %v", err)
	}
}

func TestSchoolAndRoleBoundaries(t *testing.T) {
	ctx := context.Background()
	s := fixture(t)
	northPassword, southPassword, memberPassword := secret(t), secret(t), secret(t)
	const adminEmail = "admin@example.test"
	const memberEmail = "member@example.test"
	north, err := s.Seed(ctx, "north", adminEmail, northPassword)
	if err != nil {
		t.Fatal(err)
	}
	south, err := s.Seed(ctx, "south", adminEmail, southPassword)
	if err != nil {
		t.Fatal(err)
	}
	asAdmin := login(t, s, "north", adminEmail, northPassword)
	if _, err := s.Login(ctx, "south", adminEmail, northPassword); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("password crossed schools: %v", err)
	}
	if err := asAdmin.Principal.Require("south", access.WorkspaceWrite); !errors.Is(err, access.ErrForbidden) {
		t.Fatalf("cross-school write allowed: %v", err)
	}
	if _, err := s.CreateMember(ctx, asAdmin.Token, "south", memberEmail, memberPassword, models.RoleViewer); !errors.Is(err, access.ErrForbidden) {
		t.Fatalf("cross-school grant allowed: %v", err)
	}
	if _, err := s.CreateMember(ctx, asAdmin.Token, "north", memberEmail, memberPassword, models.RoleAdmin); !errors.Is(err, access.ErrForbidden) {
		t.Fatalf("created another bootstrap admin: %v", err)
	}
	member, err := s.CreateMember(ctx, asAdmin.Token, "north", memberEmail, memberPassword, models.RoleViewer)
	if err != nil {
		t.Fatal(err)
	}
	asMember := login(t, s, "north", memberEmail, memberPassword)
	if !asMember.Principal.Allows(access.WorkspaceRead) || asMember.Principal.Allows(access.WorkspaceWrite) {
		t.Fatal("incorrect viewer privileges")
	}
	if err := s.SetMemberRole(ctx, asMember.Token, "north", member.ID, models.RoleRegistrar); !errors.Is(err, access.ErrForbidden) {
		t.Fatalf("self-promotion allowed: %v", err)
	}
	if err := s.SetMemberRole(ctx, asAdmin.Token, "north", south.ID, models.RoleViewer); !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign member exposed: %v", err)
	}
	if err := s.SetMemberRole(ctx, asAdmin.Token, "north", north.ID, models.RoleViewer); !errors.Is(err, access.ErrForbidden) {
		t.Fatalf("admin modified: %v", err)
	}
	if err := s.SetMemberRole(ctx, asAdmin.Token, "north", member.ID, models.RoleRegistrar); err != nil {
		t.Fatal(err)
	}
	p, err := s.Authenticate(ctx, asMember.Token)
	if err != nil || !p.Allows(access.WorkspaceWrite) || p.Allows(access.ManageMembers) {
		t.Fatalf("unexpected registrar permissions: %v", err)
	}
	if err := s.SetMemberRole(ctx, asAdmin.Token, "north", member.ID, models.RoleViewer); err != nil {
		t.Fatal(err)
	}
	p, err = s.Authenticate(ctx, asMember.Token)
	if err != nil || p.Allows(access.WorkspaceWrite) {
		t.Fatalf("revoked write grant remained active: %v", err)
	}
	if _, err := s.pool.DB.ExecContext(ctx, "DELETE FROM auth_memberships WHERE school_id=? AND user_id=?", "north", member.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, asMember.Token); !errors.Is(err, access.ErrUnauthenticated) {
		t.Fatalf("removed member kept a valid token: %v", err)
	}
}

func TestRejectInvalidPasswordLength(t *testing.T) {
	s := fixture(t)
	for _, invalid := range []string{"", strings.Repeat("a", 11), strings.Repeat("a", 73)} {
		if _, err := s.Seed(t.Context(), "north", "admin@example.test", invalid); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("accepted password of length %d", len(invalid))
		}
	}
}
