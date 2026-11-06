package edumigrations_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/elcomware/edupilot-os/app/facades"
	"github.com/elcomware/edupilot-os/bootstrap"
	"github.com/elcomware/edupilot-os/modules"
	"github.com/elcomware/edupilot-os/modules/documents/adapters/outbound/blob"
	files "github.com/elcomware/edupilot-os/modules/documents/application"
	"github.com/elcomware/edupilot-os/platform/access"
	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/elcomware/edupilot-os/platform/identity"
	goraveldatabase "github.com/goravel/framework/database"
	"github.com/goravel/framework/database/migration"
	"github.com/goravel/framework/log"
	"github.com/goravel/postgres"
	"github.com/goravel/sqlite"
	"github.com/jackc/pgx/v5"
)

type starterSchool struct{ School, User, Password, Hash, Token string }
type starterEvent struct{ ID, School, State, Payload string }

type starterBlob struct {
	School, ID, State string
	Content, Record   []byte
}

// StarterFixture is source evidence only; it neither upgrades nor activates v2.
// Handles and references stay available to subsequent tests in this package.
type StarterFixture struct {
	Control    database.Pool
	Shared     database.Pool
	Dedicated  map[string]database.Pool
	Catalog    *database.Catalog
	Routes     map[string]database.Pool
	Schools    []starterSchool
	Blobs      []starterBlob
	Events     []starterEvent
	BlobRoot   string
	RecordID   string
	Signatures []string
}

func starterFixture(t *testing.T, mode string) StarterFixture {
	t.Helper()
	return starterFixtureDriver(t, mode, "sqlite")
}

func starterFixtureDriver(t *testing.T, mode, driver string) StarterFixture {
	t.Helper()
	if _, err := os.ReadFile("testdata/starter/README.md"); err != nil {
		t.Fatalf("starter fixture documentation required: %v", err)
	}
	data, err := os.ReadFile("testdata/starter/schools.json")
	if err != nil {
		t.Fatalf("starter source fixture missing: %v", err)
	}
	f := StarterFixture{Dedicated: map[string]database.Pool{}, Routes: map[string]database.Pool{}, BlobRoot: filepath.Join(t.TempDir(), "blobs"), RecordID: "11111111-1111-4111-8111-111111111111"}
	if err := json.Unmarshal(data, &f.Schools); err != nil {
		t.Fatal(err)
	}
	if len(f.Schools) != 2 {
		t.Fatal("fixture requires two legacy schools")
	}
	settings := database.Settings{Control: database.Connection{Name: driver, Driver: driver, Database: filepath.Join(t.TempDir(), "control.sqlite")}}
	switch mode {
	case "shared":
	case "dedicated", "mixed":
		for i, s := range f.Schools {
			if mode == "mixed" && i == 0 {
				continue
			}
			settings.Dedicated = append(settings.Dedicated, database.Connection{Name: "tenant_" + s.School, Tenant: s.School, Driver: driver, Database: filepath.Join(t.TempDir(), s.School+".sqlite")})
		}
	default:
		t.Fatalf("unknown fixture placement %s", mode)
	}
	if driver == "postgres" {
		settings.Control.Host, settings.Control.Port, settings.Control.Username, settings.Control.Database, settings.Control.SSLMode = "localhost", 5432, "fixture", "fixture_control", "disable"
		for i := range settings.Dedicated {
			n := &settings.Dedicated[i]
			n.Host, n.Port, n.Username, n.Database, n.SSLMode = "localhost", 5432, "fixture", "fixture_"+n.Tenant, "disable"
		}
	}
	t.Setenv("DB_CONNECTION", driver)
	t.Setenv("DB_DATABASE", settings.Control.Database)
	if driver == "postgres" {
		t.Setenv("DB_HOST", "localhost")
		t.Setenv("DB_PORT", "5432")
		t.Setenv("DB_USERNAME", "fixture")
		t.Setenv("DB_SSLMODE", "disable")
	}
	dedicatedJSON, err := json.Marshal(settings.Dedicated)
	if err != nil {
		t.Fatal(err)
	}
	if settings.Dedicated == nil {
		dedicatedJSON = []byte("[]")
	}
	t.Setenv("EDUPILOT_DEDICATED_DATABASES", string(dedicatedJSON))
	connections := settings.Connections()
	postgresTargets := map[string]starterPostgresTarget{}
	if driver == "postgres" {
		for _, node := range settings.Nodes() {
			// Each handle and native migration uses its own freshly created schema;
			// public and any caller-supplied search_path are never modified.
			isolated := postgresPool(t, os.Getenv("EDUPILOT_TEST_POSTGRES_DSN"))
			var databaseName, schema string
			if err := isolated.QueryRow("SELECT current_database(),current_schema()").Scan(&databaseName, &schema); err != nil {
				t.Fatal(err)
			}
			postgresTargets[node.Name] = starterPostgresTarget{Database: databaseName, Schema: schema}
			dsn, err := starterSchemaDSN(os.Getenv("EDUPILOT_TEST_POSTGRES_DSN"), schema)
			if err != nil {
				t.Fatal(err)
			}
			if err := starterPostgresConfig(connections[node.Name].(map[string]any), dsn, schema); err != nil {
				t.Fatal(err)
			}
		}
	}
	app := facades.App()
	// Each native migrator specification owns its provider registry and SQL pools.
	app.Fresh()
	t.Cleanup(func() { app.Fresh() })
	facades.Config().Add("database", map[string]any{"default": driver, "connections": connections, "pool": map[string]any{"max_open_conns": 3, "max_idle_conns": 3}})
	(&log.ServiceProvider{}).Register(app)
	(&sqlite.ServiceProvider{}).Register(app)
	(&postgres.ServiceProvider{}).Register(app)
	(&goraveldatabase.ServiceProvider{}).Register(app)
	schema := facades.Schema()
	// Use the composition root itself: jobs/failed_jobs, native ledger, original
	// registered plans, target receipts and connection suffixes are all real.
	snapshot := loadStarterRegistry(t)
	selected, err := validateStarterRegistry(snapshot, settings, bootstrap.Migrations())
	if err != nil {
		t.Fatal(err)
	}
	schema.Register(selected)
	catalog, err := database.Open(settings, func(node database.Connection) (*sql.DB, error) { return facades.Orm().Connection(node.Name).DB() })
	if err != nil {
		t.Fatal(err)
	}
	f.Catalog, f.Control, f.Shared = catalog, catalog.Control, catalog.Control
	for _, p := range catalog.All() {
		db := p.DB
		t.Cleanup(func() { db.Close() })
	}
	for _, n := range settings.Dedicated {
		f.Dedicated[n.Tenant] = catalog.Named()[n.Name]
	}
	migrator := migration.NewMigrator(nil, schema, "migrations")
	migrate := func() error { return migrator.Run() }
	if driver == "postgres" {
		// Inspect the exact Goravel handles used by native DDL, not only the admin
		// helpers that allocated schemas. Keep one verified connection per pool.
		for _, node := range settings.Nodes() {
			actual, err := facades.Orm().Connection(node.Name).DB()
			if err != nil {
				t.Fatal(err)
			}
			if actual != catalog.Named()[node.Name].DB {
				t.Fatal("native PostgreSQL handle changed before migration")
			}
			actual.SetMaxOpenConns(1)
			actual.SetMaxIdleConns(1)
			if err := starterBeforePostgresDDL(t.Context(), postgresTargets[node.Name], postgresHandleIdentity(actual), func() error { return nil }); err != nil {
				t.Fatal(err)
			}
		}
		control, err := schema.Orm().DB()
		if err != nil {
			t.Fatal(err)
		}
		if control != f.Control.DB {
			t.Fatal("native schema ORM uses an unverified PostgreSQL handle")
		}
		if err := starterBeforePostgresDDL(t.Context(), postgresTargets[settings.Control.Name], postgresHandleIdentity(control), migrate); err != nil {
			t.Fatal(err)
		}
	} else if err := migrate(); err != nil {
		t.Fatal(err)
	}
	for _, m := range schema.Migrations() {
		f.Signatures = append(f.Signatures, m.Signature())
	}
	// Build legacy source routing directly from original pool markers. Evolving
	// runtime routing/readiness must not provision or reinterpret starter data.
	for _, s := range f.Schools {
		pool := f.Control
		if dedicated, ok := f.Dedicated[s.School]; ok {
			pool = dedicated
		}
		var owner, poolID string
		if err := pool.DB.QueryRow("SELECT owner,pool_id FROM edu_pool_marker WHERE singleton=1").Scan(&owner, &poolID); err != nil {
			t.Fatal(err)
		}
		expectedOwner := "shared"
		if pool.DB != f.Control.DB {
			expectedOwner = "tenant:" + s.School
		}
		if owner != expectedOwner || poolID == "" {
			t.Fatal("legacy pool marker mismatch")
		}
		if _, err := f.Control.DB.Exec(f.Control.Bind("INSERT INTO edu_tenant_placement(tenant_id,pool_id) VALUES (?,?)"), s.School, poolID); err != nil {
			t.Fatal(err)
		}
		f.Routes[s.School] = pool
	}
	for _, s := range f.Schools {
		pool := f.Routes[s.School]
		exec := func(p database.Pool, q string, args ...any) {
			t.Helper()
			if _, err := p.DB.Exec(p.Bind(q), args...); err != nil {
				t.Fatalf("seed %s: %v", s.School, err)
			}
		}
		exec(f.Control, "INSERT INTO identity_schools VALUES (?)", s.School)
		exec(f.Control, "INSERT INTO identity_accounts VALUES (?,?,?,?)", s.School, s.User, "same@example.test", s.Hash)
		exec(f.Control, "INSERT INTO identity_memberships VALUES (?,?,?)", s.School, s.User, "admin")
		token, err := base64.RawURLEncoding.DecodeString(s.Token)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(token)
		exec(f.Control, "INSERT INTO identity_sessions VALUES (?,?,?,?)", hex.EncodeToString(sum[:]), s.School, s.User, int64(4102444800))
		exec(pool, "INSERT INTO students_records VALUES (?,?,?,?,?,?)", s.School, f.RecordID, "Élève "+s.School, "Collision", "active", 3)
		exec(pool, "INSERT INTO students_records VALUES (?,?,?,?,?,?)", s.School, "archived-student", "Former", "Student", "archived", 5)
		exec(pool, "INSERT INTO enrollment_placements VALUES (?,?,?,?,?,?,?,?)", s.School, "placement-retry", f.RecordID, "2026-2027", "Grade 7", "Science", "withdrawn", 2)
		for _, d := range modules.Definitions() {
			if d.Migration != nil {
				exec(pool, "INSERT INTO ws_"+d.Key+" VALUES (?,?,?,?,?)", s.School, f.RecordID, d.Key+" private "+s.School, "closed", 4)
			}
		}
		payload := `{"studentId":"` + f.RecordID + `","version":3,"status":"active","actorId":"` + s.User + `"}`
		for _, state := range []string{"pending", "dead", "published"} {
			var lease, until, published, dead, last any
			attempts := 0
			switch state {
			case "leased":
				lease = "legacy-lease-" + s.School
				until = int64(4102444800000)
				attempts = 2
			case "dead":
				dead = int64(1760000000000)
				last = "fixture delivery failure"
				attempts = 8
			case "published":
				published = int64(1760000000000)
				attempts = 3
			}
			eventID := s.School + "-" + state
			f.Events = append(f.Events, starterEvent{ID: eventID, School: s.School, State: state, Payload: payload})
			exec(pool, "INSERT INTO edu_outbox VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)", eventID, s.School, "students.registered.v1", f.RecordID, payload, int64(1759999999000), int64(1760000000000), attempts, lease, until, published, dead, last)
			if state == "published" {
				exec(pool, "INSERT INTO edu_audit_inbox VALUES (?,?,?,?,?)", eventID, s.School, "students.registered.v1", f.RecordID, int64(1760000000000))
			}
		}
	}
	// Checked-in legacy bytes are independent of the decoder under test, with
	// stable IDs and paths. A missing object remains a reference without bytes.
	objectData, err := os.ReadFile("testdata/starter/objects.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(objectData, &f.Blobs); err != nil {
		t.Fatal(err)
	}
	allBlobs := f.Blobs
	f.Blobs = nil
	for _, b := range allBlobs {
		if b.State != "valid" {
			continue
		}
		f.Blobs = append(f.Blobs, b)
		path := filepath.Join(f.BlobRoot, "schools", b.School, "documents", b.ID)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, b.Record, 0600); err != nil {
			t.Fatal(err)
		}
	}

	return f
}

func TestStarterFixture(t *testing.T) {
	for _, mode := range []string{"shared", "dedicated", "mixed"} {
		t.Run(mode, func(t *testing.T) { verifyStarterFixture(t, starterFixture(t, mode), mode) })
	}
}
func TestStarterFixturePostgres(t *testing.T) {
	if os.Getenv("EDUPILOT_TEST_POSTGRES_DSN") == "" {
		t.Skip("EDUPILOT_TEST_POSTGRES_DSN unavailable; PostgreSQL fixture verification remains open")
	}
	for _, mode := range []string{"shared", "dedicated", "mixed"} {
		t.Run(mode, func(t *testing.T) { verifyStarterFixture(t, starterFixtureDriver(t, mode, "postgres"), mode) })
	}
}
func verifyStarterFixture(t *testing.T, f StarterFixture, mode string) {
	t.Helper()
	verifyStarterTables(t, f)
	count(t, f.Control, "SELECT count(*) FROM migrations", len(f.Signatures))
	count(t, f.Control, "SELECT count(*) FROM identity_accounts WHERE email='same@example.test'", 2)
	count(t, f.Control, "SELECT count(*) FROM edu_tenant_placement", 2)
	sharedSchools := map[string]int{"shared": 2, "mixed": 1, "dedicated": 0}[mode]
	count(t, f.Control, "SELECT count(*) FROM students_records", 2*sharedSchools)
	count(t, f.Control, "SELECT count(*) FROM enrollment_placements", sharedSchools)
	if len(f.Events) != 6 || len(f.Blobs) != 2 {
		t.Fatal("fixture event/object references incomplete")
	}
	for _, d := range modules.Definitions() {
		if d.Migration != nil {
			count(t, f.Control, "SELECT count(*) FROM ws_"+d.Key, sharedSchools)
		}
	}
	for _, table := range []string{"jobs", "failed_jobs"} {
		count(t, f.Control, "SELECT count(*) FROM "+table, 0)
	}
	expectedDedicated := map[string]int{"shared": 0, "mixed": 1, "dedicated": 2}[mode]
	if len(f.Dedicated) != expectedDedicated {
		t.Errorf("dedicated handles = %d", len(f.Dedicated))
	}
	for _, s := range f.Schools {
		p := f.Routes[s.School]
		if _, dedicated := f.Dedicated[s.School]; dedicated == (p.DB == f.Shared.DB) {
			t.Fatal("fixture placement mismatch")
		}
		if p.DB != f.Control.DB {
			var got int
			query := "SELECT count(*) FROM sqlite_master WHERE name IN ('identity_accounts','migrations','jobs','failed_jobs')"
			if p.Driver == "postgres" {
				query = "SELECT count(*) FROM information_schema.tables WHERE table_schema=current_schema() AND table_name IN ('identity_accounts','migrations','jobs','failed_jobs')"
			}
			if err := p.DB.QueryRow(query).Scan(&got); err != nil || got != 0 {
				t.Fatalf("control data leaked to dedicated pool: %d %v", got, err)
			}
		}
		q := func(query string, want int, args ...any) {
			t.Helper()
			var got int
			if err := p.DB.QueryRow(p.Bind(query), args...).Scan(&got); err != nil || got != want {
				t.Fatalf("%s: got %d want %d: %v", query, got, want, err)
			}
		}
		q("SELECT count(*) FROM students_records WHERE school_id=? AND id=? AND version=3", 1, s.School, f.RecordID)
		q("SELECT count(*) FROM students_records WHERE school_id=? AND status='archived' AND version=5", 1, s.School)
		q("SELECT count(*) FROM enrollment_placements WHERE school_id=? AND id='placement-retry' AND status='withdrawn' AND version=2", 1, s.School)
		// The original uniqueness constraint must reject a retry with a different ID.
		if _, err := p.DB.Exec(p.Bind("INSERT INTO enrollment_placements SELECT school_id,'retry-other-id',student_id,academic_year,grade,course,status,version FROM enrollment_placements WHERE school_id=?"), s.School); err == nil {
			t.Fatal("starter placement retry uniqueness lost")
		}
		for _, d := range modules.Definitions() {
			if d.Migration != nil {
				q("SELECT count(*) FROM ws_"+d.Key+" WHERE school_id=? AND id=? AND version=4", 1, s.School, f.RecordID)
			}
		}
		q("SELECT count(*) FROM edu_outbox WHERE tenant_id=?", 3, s.School)
		q("SELECT count(*) FROM edu_outbox WHERE tenant_id=? AND dead_at IS NOT NULL AND attempts=8", 1, s.School)
		q("SELECT count(*) FROM edu_outbox WHERE tenant_id=? AND lease_id IS NOT NULL", 0, s.School)
		q("SELECT count(*) FROM edu_outbox WHERE tenant_id=? AND published_at IS NOT NULL AND attempts=3", 1, s.School)
		q("SELECT count(*) FROM edu_outbox WHERE tenant_id=? AND published_at IS NULL AND dead_at IS NULL AND lease_id IS NULL AND attempts=0", 1, s.School)
		q("SELECT count(*) FROM edu_audit_inbox WHERE tenant_id=?", 1, s.School)
		var payload string
		if err := p.DB.QueryRow(p.Bind("SELECT payload FROM edu_outbox WHERE event_id=?"), s.School+"-pending").Scan(&payload); err != nil {
			t.Fatal(err)
		}
		expected := `{"studentId":"` + f.RecordID + `","version":3,"status":"active","actorId":"` + s.User + `"}`
		if payload != expected {
			t.Fatal("legacy payload bytes changed")
		}

	}
	for _, p := range f.Catalog.All() {
		var receipts int
		if err := p.DB.QueryRow("SELECT count(*) FROM edu_migration_receipts").Scan(&receipts); err != nil {
			t.Fatal(err)
		}
		expected := 0
		for _, sig := range f.Signatures {
			if strings.HasSuffix(sig, "__"+poolName(f, p)) {
				expected++
			}
		}
		if receipts != expected {
			t.Fatalf("receipt coverage %d want %d", receipts, expected)
		}
	}
}

func poolName(f StarterFixture, p database.Pool) string {
	if p.DB == f.Control.DB {
		return p.Driver
	}
	for school, node := range f.Dedicated {
		if node.DB == p.DB {
			return "tenant_" + school
		}
	}
	return "unknown"
}

// ConnString preserves original input, so modifying pgx RuntimeParams alone is
// insufficient when passing a DSN on to another parser (the native driver).
func starterSchemaDSN(dsn, schema string) (string, error) {
	if _, err := pgx.ParseConfig(dsn); err != nil {
		return "", err
	}
	if !regexp.MustCompile(`^edu_[a-z0-9_]+$`).MatchString(schema) {
		return "", errors.New("invalid isolated fixture schema")
	}
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		parsed, err := url.Parse(dsn)
		if err != nil {
			return "", err
		}
		query := parsed.Query()
		query.Set("search_path", schema)
		parsed.RawQuery = query.Encode()
		return parsed.String(), nil
	}
	return dsn + " search_path=" + schema, nil
}
func TestStarterFixtureSchemaDSN(t *testing.T) {
	for _, dsn := range []string{"postgres://fixture:password@localhost:5432/db?sslmode=disable&search_path=public", "host=localhost user=fixture password=password dbname=db search_path=public sslmode=disable"} {
		got, err := starterSchemaDSN(dsn, "edu_isolated")
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := pgx.ParseConfig(got)
		if err != nil {
			t.Fatal(err)
		}
		if parsed.RuntimeParams["search_path"] != "edu_isolated" || parsed.Password != "password" {
			t.Errorf("native driver DSN lost schema isolation or connection credentials")
		}
	}
}

func starterPostgresConfig(connection map[string]any, dsn, schema string) error {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return err
	}
	connection["dsn"], connection["schema"], connection["database"] = dsn, schema, cfg.Database
	connection["host"], connection["port"], connection["username"], connection["password"] = cfg.Host, int(cfg.Port), cfg.User, cfg.Password
	return nil
}
func TestStarterFixturePostgresConfig(t *testing.T) {
	connection := map[string]any{"schema": "public", "database": "placeholder"}
	dsn, err := starterSchemaDSN("postgres://fixture:password@localhost:5432/actual?sslmode=disable", "edu_isolated")
	if err != nil {
		t.Fatal(err)
	}
	if err := starterPostgresConfig(connection, dsn, "edu_isolated"); err != nil {
		t.Fatal(err)
	}
	if connection["schema"] != "edu_isolated" || connection["database"] != "actual" {
		t.Fatal("native schema introspection uses wrong schema/database")
	}
}

func verifyStarterCompatibilityReaders(t *testing.T, f StarterFixture) {
	for _, s := range f.Schools {
		// Existing session authenticates the exact legacy school/user. Equal emails
		// have independent credentials; the other school's password never works.
		service := identity.New(&f.Control)
		principal, err := service.Authenticate(t.Context(), s.Token)
		if err != nil || principal.ID != s.User || principal.School != s.School {
			t.Fatalf("legacy session: %v", err)
		}
		session, err := service.Login(t.Context(), s.School, "same@example.test", s.Password)
		if err != nil || session.Principal.ID != s.User {
			t.Fatalf("legacy password hash: %v", err)
		}
		for _, other := range f.Schools {
			if other.School != s.School {
				if _, err := service.Login(t.Context(), s.School, "same@example.test", other.Password); !errors.Is(err, identity.ErrInvalidCredentials) {
					t.Fatal("credentials merged across equal email")
				}
			}
		}
	}
	store, err := blob.Configured(func(k, fallback string) string {
		if k == "EDUPILOT_DOCUMENTS_ROOT" {
			return f.BlobRoot
		}
		return fallback
	})
	if err != nil {
		t.Fatal(err)
	}
	service := files.Files{Resolve: func() (files.BlobStore, error) { return store, nil }}
	for _, b := range f.Blobs {
		ctx := access.WithPrincipal(context.Background(), access.Principal{ID: "fixture-reader", School: b.School, Permissions: []string{access.DocumentsRead}})
		got, err := service.Download(ctx, b.ID)
		if b.State == "valid" {
			if err != nil || !bytes.Equal(got.Content, b.Content) {
				t.Fatalf("valid private blob: %v", err)
			}
			continue
		}
		want := files.ErrFileIntegrity
		if b.State == "missing" {
			want = files.ErrFileMissing
		}
		if !errors.Is(err, want) {
			t.Errorf("blob %s: %v want %v", b.State, err, want)
		}
	}
}
func starterFixtureFaults(t *testing.T, mode string) StarterFixture {
	return injectStarterFaults(t, starterFixture(t, mode))
}
func injectStarterFaults(t *testing.T, f StarterFixture) StarterFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/starter/objects.json")
	if err != nil {
		t.Fatal(err)
	}
	var objects []starterBlob
	if err := json.Unmarshal(data, &objects); err != nil {
		t.Fatal(err)
	}
	for _, b := range objects {
		if b.State == "valid" {
			continue
		}
		f.Blobs = append(f.Blobs, b)
		if b.State == "missing" {
			continue
		}
		path := filepath.Join(f.BlobRoot, "schools", b.School, "documents", b.ID)
		if err := os.WriteFile(path, b.Record, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, s := range f.Schools {
		pool := f.Routes[s.School]
		id := s.School + "-leased"
		if _, err := pool.DB.Exec(pool.Bind("INSERT INTO edu_outbox SELECT ?,tenant_id,event_type,aggregate_id,payload,created_at,available_at,2,?,4102444800000,NULL,NULL,NULL FROM edu_outbox WHERE event_id=?"), id, "legacy-lease-"+s.School, s.School+"-pending"); err != nil {
			t.Fatal(err)
		}
		f.Events = append(f.Events, starterEvent{ID: id, School: s.School, State: "leased"})
	}
	return f
}
func TestStarterFixtureFaults(t *testing.T) {
	for _, mode := range []string{"shared", "dedicated", "mixed"} {
		t.Run(mode, func(t *testing.T) { verifyStarterFaults(t, starterFixtureFaults(t, mode)) })
	}
}
func verifyStarterFaults(t *testing.T, f StarterFixture) {
	t.Helper()
	if len(f.Blobs) != 11 || len(f.Events) != 8 {
		t.Fatal("fault profile references incomplete")
	}
	for _, s := range f.Schools {
		p := f.Routes[s.School]
		var leases int
		if err := p.DB.QueryRow(p.Bind("SELECT count(*) FROM edu_outbox WHERE tenant_id=? AND lease_until=4102444800000 AND attempts=2"), s.School).Scan(&leases); err != nil || leases != 1 {
			t.Fatalf("lease fault missing: %d %v", leases, err)
		}
	}
	for _, b := range f.Blobs {
		path := filepath.Join(f.BlobRoot, "schools", b.School, "documents", b.ID)
		data, err := os.ReadFile(path)
		if b.State == "missing" {
			if !os.IsNotExist(err) {
				t.Fatal("missing object fault absent")
			}
			continue
		}
		if err != nil || !bytes.Equal(data, b.Record) {
			t.Fatalf("fault object bytes differ: %s %v", b.State, err)
		}
	}
}
func TestStarterFixtureCompatibilityReaders(t *testing.T) {
	verifyStarterCompatibilityReaders(t, starterFixtureFaults(t, "shared"))
}
