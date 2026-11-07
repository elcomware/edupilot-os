package feature

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/bootstrap"
	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// Each subprocess starts Goravel with its own throwaway connection configuration.
func TestNativeFoundation(t *testing.T) {
	if os.Getenv("EDUPILOT_FOUNDATION_CHILD") == "1" {
		exerciseFoundation(t)
		return
	}
	for _, engine := range []string{"sqlite", "postgres"} {
		t.Run(engine, func(t *testing.T) {
			var ctrl, ded, candidate, hierarchy database.Connection
			if engine == "sqlite" {
				root := t.TempDir()
				ctrl = database.Connection{Driver: engine, Database: filepath.Join(root, "control.sqlite")}
				ded = database.Connection{Driver: engine, Database: filepath.Join(root, "west.sqlite"), Tenant: "west"}
				candidate = database.Connection{Driver: engine, Database: filepath.Join(root, "candidate.sqlite")}
				hierarchy = database.Connection{Driver: engine, Database: filepath.Join(root, "hierarchy.sqlite")}
			} else {
				dsn := os.Getenv("EDUPILOT_TEST_POSTGRES_DSN")
				if dsn == "" || os.Getenv("EDUPILOT_TEST_POSTGRES_CREATE_DATABASES") != "1" {
					t.Skip("requires disposable PostgreSQL with CREATE DATABASE")
				}
				cfg, err := pgx.ParseConfig(dsn)
				if err != nil {
					t.Fatal("invalid PostgreSQL fixture configuration")
				}
				u, err := url.Parse(dsn)
				if err != nil {
					t.Fatal("invalid PostgreSQL fixture URL")
				}
				admin := stdlib.OpenDB(*cfg)
				t.Cleanup(func() { admin.Close() })
				makeDB := func() database.Connection {
					name := "edufoundation_" + strings.ToLower(rand.Text())
					quoted := pgx.Identifier{name}.Sanitize()
					if _, err := admin.ExecContext(t.Context(), "CREATE DATABASE "+quoted+" TEMPLATE template0"); err != nil {
						t.Fatal("cannot create fixture database")
					}
					t.Cleanup(func() {
						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()
						if _, err := admin.ExecContext(ctx, "DROP DATABASE "+quoted); err != nil {
							t.Error("fixture cleanup failed")
						}
					})
					return database.Connection{Driver: "postgres", Database: name, Username: cfg.User, Password: cfg.Password, Host: cfg.Host, Port: int(cfg.Port), SSLMode: u.Query().Get("sslmode")}
				}
				ctrl = makeDB()
				ded = makeDB()
				ded.Tenant = "west"
				candidate = makeDB()
				hierarchy = makeDB()
			}
			candidate.Tenant = candidateID
			hierarchy.Tenant = hierarchyTenantID
			placement, _ := json.Marshal([]database.Connection{ded, candidate, hierarchy})
			bin, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.CommandContext(t.Context(), bin, "-test.run=^TestNativeFoundation$", "-test.v")
			cmd.Dir = t.TempDir()
			cmd.Env = append(os.Environ(), "APP_KEY="+strings.Repeat("a", 32), "APP_ENV=testing", "EDUPILOT_FOUNDATION_CHILD=1", "DB_CONNECTION="+engine, "DB_DATABASE="+ctrl.Database, "DB_HOST="+ctrl.Host, "DB_PORT="+strconv.Itoa(ctrl.Port), "DB_USERNAME="+ctrl.Username, "DB_PASSWORD="+ctrl.Password, "DB_SSLMODE="+ctrl.SSLMode, "EDUPILOT_DEDICATED_DATABASES="+string(placement), "EDUPILOT_ENABLED_MODULES=tenancy,auth")
			if output, err := cmd.CombinedOutput(); err != nil {
				if ctrl.Password != "" {
					output = bytes.ReplaceAll(output, []byte(ctrl.Password), []byte("[redacted]"))
				}
				t.Fatalf("foundation drill failed: %v\n%s", err, output)
			}
		})
	}
}

func exerciseFoundation(t *testing.T) {
	bootstrap.Boot()
	db, err := database.OpenConfigured()
	if err != nil {
		t.Fatal("open pool catalog")
	}
	ctx := t.Context()
	if db.Control.Driver == "sqlite" {
		// The SQLite-reserved prefix includes the underscore. A similarly named
		// application table must cause rejection before the ledger is created.
		if _, err := db.Control.DB.Exec("CREATE TABLE sqlitex_records (id INTEGER)"); err != nil {
			t.Fatal(err)
		}
		schemaVersion := func() int {
			t.Helper()
			var version int
			if err := db.Control.DB.QueryRow("PRAGMA schema_version").Scan(&version); err != nil {
				t.Fatal(err)
			}
			return version
		}
		before := schemaVersion()
		if err := edumigrations.Run(ctx, db); err == nil {
			t.Fatal("accepted a control database with an unrelated application table")
		}
		if after := schemaVersion(); after != before {
			t.Fatalf("installer modified the rejected control schema: version %d became %d", before, after)
		}
		if _, err := db.Control.DB.Exec("DROP TABLE sqlitex_records"); err != nil {
			t.Fatal(err)
		}
	}
	// Refuse an unrelated dedicated database before writing to the control pool.
	west := db.Named()["tenant_west"]
	if _, err := west.DB.Exec("CREATE TABLE occupied (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if err := edumigrations.Run(ctx, db); err == nil {
		t.Fatal("accepted an occupied target")
	}
	if _, err := db.Control.DB.Exec("SELECT 1 FROM migrations"); err == nil {
		t.Fatal("modified control before rejecting dedicated pool")
	}
	if _, err := west.DB.Exec("DROP TABLE occupied"); err != nil {
		t.Fatal(err)
	}
	// A view can be the only application object and can even use the ledger name.
	// Neither pool may be claimed until its existing schema is recognized.
	for _, pool := range db.All() {
		if _, err := pool.DB.Exec("CREATE VIEW migrations AS SELECT 1 AS batch WHERE 1=0"); err != nil {
			t.Fatal(err)
		}
		if err := edumigrations.Run(ctx, db); err == nil {
			t.Fatal("accepted a database containing a view named migrations")
		}
		if _, err := db.Control.DB.Exec("SELECT 1 FROM edu_foundation_receipts"); err == nil {
			t.Fatal("installer claimed a pool before rejecting the view")
		}
		if _, err := pool.DB.Exec("DROP VIEW migrations"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Control.DB.Exec("SELECT 1 FROM migrations"); err == nil {
			t.Fatal("installer created a ledger before rejecting the view")
		}
	}
	for i := 0; i < 2; i++ {
		if err := edumigrations.Run(ctx, db); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range db.All() {
		for _, name := range []string{"ws_tenancy", "tenancy_campuses", "edu_outbox"} {
			if _, err := p.DB.Exec("SELECT 1 FROM " + name); err != nil {
				t.Fatal(name, err)
			}
		}
		for _, name := range []string{"ws_students", "ws_auth", "students_records"} {
			if _, err := p.DB.Exec("SELECT 1 FROM " + name); err == nil {
				t.Fatal("unexpected table", name)
			}
		}
	}
	if _, err := west.DB.Exec("SELECT 1 FROM auth_users"); err == nil {
		t.Fatal("credentials present in data pool")
	}
	verifyHierarchyConstraints(t, db.Control)
	verifyPersistedDirectory(t, db)
	verifyOrganizationWriter(t, db)
	verifySharedProvisioning(t, db)
	verifyDedicatedCandidate(t, db)
	verifyDedicatedReservation(t, db)
	verifyDedicatedPreparation(t, db)
	verifyDedicatedActivation(t, db)
	verifyDedicatedEnrollment(t, db)
	t.Run("hierarchy_shared", func(t *testing.T) { verifySchoolCampusWriters(t, db, requireValue(scope.NewID[scope.Tenant]())) })
	t.Run("hierarchy_dedicated", func(t *testing.T) {
		verifySchoolCampusWriters(t, db, requireValue(scope.Parse[scope.Tenant](hierarchyTenantID)))
	})
	t.Run("directory_shared", func(t *testing.T) { verifyHierarchyReader(t, db, requireValue(scope.NewID[scope.Tenant]())) })
	t.Run("directory_dedicated", func(t *testing.T) {
		verifyHierarchyReader(t, db, requireValue(scope.Parse[scope.Tenant](hierarchyTenantID)))
	})
	t.Run("metadata_shared", func(t *testing.T) { verifyHierarchyEditor(t, db, requireValue(scope.NewID[scope.Tenant]())) })
	t.Run("metadata_dedicated", func(t *testing.T) {
		verifyHierarchyEditor(t, db, requireValue(scope.Parse[scope.Tenant](hierarchyTenantID)))
	})
	// A recognized receipt table still has to belong to the configured tenant.
	if _, err := west.DB.Exec("UPDATE edu_pool_marker SET owner='tenant:other'"); err != nil {
		t.Fatal(err)
	}
	if err := edumigrations.Run(ctx, db); err == nil {
		t.Fatal("installer accepted a different tenant's pool")
	}
	if _, err := west.DB.Exec("UPDATE edu_pool_marker SET owner='tenant:west'"); err != nil {
		t.Fatal(err)
	}
	// Interruption between the dedicated commit and the control ledger write.
	const key = "20261106_foundation_05_workspace_tenancy__tenant_west"
	if _, err := db.Control.DB.Exec(db.Control.Bind("DELETE FROM migrations WHERE migration=?"), key); err != nil {
		t.Fatal(err)
	}
	// A restored dedicated pool can carry the same physical ID as the shared
	// pool even when its owner label is correct. Reject it before repairing
	// the pending control ledger entry.
	var dedicatedID, sharedID string
	for _, marker := range []struct {
		pool database.Pool
		id   *string
	}{{west, &dedicatedID}, {db.Control, &sharedID}} {
		if err := marker.pool.DB.QueryRow("SELECT pool_id FROM edu_pool_marker WHERE singleton=1").Scan(marker.id); err != nil {
			t.Fatal(err)
		}
	}
	setDedicatedID := func(id string) {
		t.Helper()
		if _, err := west.DB.Exec(west.Bind("UPDATE edu_pool_marker SET pool_id=? WHERE singleton=1"), id); err != nil {
			t.Fatal(err)
		}
	}
	setDedicatedID(sharedID)
	if err := db.CheckReady(ctx); !errors.Is(err, database.ErrPlacement) {
		t.Fatalf("readiness accepted duplicate physical IDs: %v", err)
	}
	if err := edumigrations.Run(ctx, db); !errors.Is(err, database.ErrPlacement) {
		t.Fatalf("installer accepted duplicate physical IDs: %v", err)
	}
	var recorded int
	if err := db.Control.DB.QueryRow(db.Control.Bind("SELECT count(*) FROM migrations WHERE migration=?"), key).Scan(&recorded); err != nil || recorded != 0 {
		t.Fatalf("installer advanced pending migration on ambiguous placement: %d, %v", recorded, err)
	}
	setDedicatedID(dedicatedID)
	if err := edumigrations.Run(ctx, db); err != nil {
		t.Fatal("retry after pool commit", err)
	}
	svc := services.New(&db.Control)
	pw := rand.Text()
	if _, err = svc.Seed(ctx, "west", "staff@example.test", pw); err != nil {
		t.Fatal(err)
	}
	active, err := svc.Login(ctx, "west", "staff@example.test", pw)
	if err != nil {
		t.Fatal(err)
	}
	get := func(token, scope string, want int) string {
		req := httptest.NewRequest("GET", "/api/v1/modules", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		if scope != "" {
			req.Header.Set("X-School-ID", scope)
		}
		w := httptest.NewRecorder()
		facades.Route().ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("directory returned %d, expected %d", w.Code, want)
		}
		return w.Body.String()
	}
	if !strings.Contains(get(active.Token, "", 200), "\"create\"") {
		t.Fatal("missing write capability")
	}
	if !facades.Cache().Has("catalog:v1:tenancy.auth:true:true") {
		t.Fatal("directory response did not populate the configured Goravel cache")
	}
	get(active.Token, "another", 403)
	if _, err := db.Control.DB.Exec("DELETE FROM auth_role_permissions WHERE role='admin' AND permission='workspace.write'"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(get(active.Token, "", 200), "\"create\"") {
		t.Fatal("catalog cache retained revoked write access")
	}
	if err := svc.Logout(ctx, active.Token); err != nil {
		t.Fatal(err)
	}
	get(active.Token, "", 401)
	if _, err := west.DB.Exec(west.Bind("DELETE FROM edu_foundation_receipts WHERE signature=?"), key); err != nil {
		t.Fatal(err)
	}
	if err := edumigrations.CheckReady(ctx, db, facades.Schema().Migrations()); err == nil {
		t.Fatal("readiness accepted a missing dedicated receipt")
	}
}
