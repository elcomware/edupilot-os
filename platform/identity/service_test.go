package identity

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

	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"golang.org/x/crypto/bcrypt"
)

func fixture(t *testing.T) *Service {
	t.Helper()
	f := url.URL{Scheme: "file", Path: filepath.Join(t.TempDir(), "identity.sqlite"), RawQuery: "_pragma=foreign_keys(ON)"}
	db, err := driver.Open(f.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	for _, stmt := range SchemaUp {
		if _, err := db.ExecContext(t.Context(), stmt); err != nil {
			t.Fatal(err)
		}
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
	if err := p.Require("north", ManageMembers); err != nil {
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
	if err := s.pool.DB.QueryRowContext(ctx, "SELECT password_hash FROM identity_accounts WHERE school_id=?", "north").Scan(&passwordHash); err != nil {
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
	if err = s.pool.DB.QueryRowContext(ctx, "SELECT token_hash FROM identity_sessions WHERE token_hash=?", hash).Scan(&stored); err != nil {
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
	if _, err := s.Authenticate(ctx, first.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("revoked token authenticated: %v", err)
	}
	now = now.Add(sessionTTL)
	if _, err := s.Authenticate(ctx, second.Token); !errors.Is(err, ErrUnauthenticated) {
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
	if err := asAdmin.Principal.Require("south", StudentsWrite); !errors.Is(err, ErrForbidden) {
		t.Fatalf("cross-school write allowed: %v", err)
	}
	if _, err := s.CreateMember(ctx, asAdmin.Token, "south", memberEmail, memberPassword, RoleViewer); !errors.Is(err, ErrForbidden) {
		t.Fatalf("cross-school grant allowed: %v", err)
	}
	if _, err := s.CreateMember(ctx, asAdmin.Token, "north", memberEmail, memberPassword, RoleAdmin); !errors.Is(err, ErrForbidden) {
		t.Fatalf("created another bootstrap admin: %v", err)
	}
	member, err := s.CreateMember(ctx, asAdmin.Token, "north", memberEmail, memberPassword, RoleViewer)
	if err != nil {
		t.Fatal(err)
	}
	asMember := login(t, s, "north", memberEmail, memberPassword)
	if !asMember.Principal.Allows(StudentsRead) || asMember.Principal.Allows(StudentsWrite) {
		t.Fatal("incorrect viewer privileges")
	}
	if err := s.SetMemberRole(ctx, asMember.Token, "north", member.ID, RoleRegistrar); !errors.Is(err, ErrForbidden) {
		t.Fatalf("self-promotion allowed: %v", err)
	}
	if err := s.SetMemberRole(ctx, asAdmin.Token, "north", south.ID, RoleViewer); !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign member exposed: %v", err)
	}
	if err := s.SetMemberRole(ctx, asAdmin.Token, "north", north.ID, RoleViewer); !errors.Is(err, ErrForbidden) {
		t.Fatalf("admin modified: %v", err)
	}
	if err := s.SetMemberRole(ctx, asAdmin.Token, "north", member.ID, RoleRegistrar); err != nil {
		t.Fatal(err)
	}
	p, err := s.Authenticate(ctx, asMember.Token)
	if err != nil || !p.Allows(StudentsWrite) || p.Allows(ManageMembers) {
		t.Fatalf("unexpected registrar permissions: %v", err)
	}
	if err := s.SetMemberRole(ctx, asAdmin.Token, "north", member.ID, RoleViewer); err != nil {
		t.Fatal(err)
	}
	p, err = s.Authenticate(ctx, asMember.Token)
	if err != nil || p.Allows(StudentsWrite) {
		t.Fatalf("revoked write grant remained active: %v", err)
	}
	if _, err := s.pool.DB.ExecContext(ctx, "DELETE FROM identity_memberships WHERE school_id=? AND user_id=?", "north", member.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, asMember.Token); !errors.Is(err, ErrUnauthenticated) {
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
