package tenantupgrade

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// This qualifies real PostgreSQL SQL/snapshots via a private opened-handle seam.
// It does not qualify the native CLI connection settings/search_path end to end.
func TestInspectionPostgresFrozenSQLReadOnlyAndDetection(t *testing.T) {
	dsn := os.Getenv("EDUPILOT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("EDUPILOT_TEST_POSTGRES_DSN unavailable; native PostgreSQL inspector SQL/read-only qualification remains open")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid disposable PostgreSQL test configuration")
	}
	admin := stdlib.OpenDB(*cfg)
	t.Cleanup(func() { admin.Close() })
	configurations := map[string]*pgx.ConnConfig{}
	writers := map[string]*sql.DB{}
	for _, name := range []string{"postgres", "tenant_west"} {
		var random [8]byte
		if _, err := rand.Read(random[:]); err != nil {
			t.Fatal(err)
		}
		schema := "edu_inspect_" + hex.EncodeToString(random[:])
		if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
				t.Error(err)
			}
		})
		scoped := cfg.Copy()
		scoped.RuntimeParams["search_path"] = schema
		writer := stdlib.OpenDB(*scoped)
		var actual string
		if err := writer.QueryRow("SELECT current_schema()").Scan(&actual); err != nil || actual != schema {
			writer.Close()
			t.Fatal("connection did not select the private PostgreSQL test schema")
		}
		writers[name] = writer
		configurations[name] = scoped
		t.Cleanup(func() { writer.Close() })
	}
	data, err := os.ReadFile("../../database/edumigrations/testdata/starter/registrations.json")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot struct {
		Registrations []struct {
			Connection, PlanID string
			Up                 []string
		}
	}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	keySet := map[string]bool{}
	for _, registration := range snapshot.Registrations {
		target := registration.Connection
		if target == "sqlite" {
			target = "postgres"
		}
		writer := writers[target]
		if writer == nil {
			continue
		}
		for _, statement := range registration.Up {
			statement = strings.ReplaceAll(statement, "'<generated>'", "'fixture-"+target+"'")
			if _, err := writer.Exec(statement); err != nil {
				t.Fatal(err)
			}
		}
		if key, ok := strings.CutPrefix(registration.PlanID, "20260102000001_workspace_"); ok {
			keySet[key] = true
		}
	}
	for _, writer := range writers {
		if _, err := writer.Exec("CREATE TABLE IF NOT EXISTS edu_migration_receipts (signature TEXT PRIMARY KEY NOT NULL)"); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Exec("INSERT INTO edu_migration_receipts VALUES ('fixture-receipt')"); err != nil {
			t.Fatal(err)
		}
	}
	for _, statement := range []string{
		"CREATE TABLE migrations (id BIGSERIAL PRIMARY KEY, migration VARCHAR(255) NOT NULL, batch INTEGER NOT NULL)",
		"CREATE TABLE jobs (id BIGSERIAL PRIMARY KEY, queue VARCHAR(255) NOT NULL, payload TEXT NOT NULL, attempts SMALLINT NOT NULL DEFAULT 0, reserved_at TIMESTAMPTZ, available_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)",
		"CREATE TABLE failed_jobs (id BIGSERIAL PRIMARY KEY, uuid VARCHAR(255) NOT NULL UNIQUE, connection TEXT NOT NULL, queue TEXT NOT NULL, payload TEXT NOT NULL, exception TEXT NOT NULL, failed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)",
	} {
		if _, err := writers["postgres"].Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	keys := []string{}
	for key := range keySet {
		keys = append(keys, key)
	}
	control := writers["postgres"]
	west := writers["tenant_west"]
	for _, school := range []string{"east", "west"} {
		marker, pool := "fixture-postgres", control
		if school == "west" {
			marker, pool = "fixture-tenant_west", west
		}
		if _, err := control.Exec("INSERT INTO identity_schools VALUES ($1)", school); err != nil {
			t.Fatal(err)
		}
		if _, err := control.Exec("INSERT INTO edu_tenant_placement VALUES ($1,$2)", school, marker); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec("INSERT INTO students_records VALUES ($1,'same-record','Private','Student','active',1)", school); err != nil {
			t.Fatal(err)
		}
		for _, key := range keys {
			if _, err := pool.Exec("INSERT INTO ws_"+key+" VALUES ($1,'same-record','Private','open',1)", school); err != nil {
				t.Fatal(err)
			}
		}
	}
	settings := database.Settings{Control: database.Connection{Name: "postgres", Driver: "postgres", Host: "localhost", Port: 5432, Username: "fixture", Database: "fixture-control", SSLMode: "disable"}, Dedicated: []database.Connection{{Name: "tenant_west", Tenant: "west", Driver: "postgres", Host: "localhost", Port: 5432, Username: "fixture", Database: "fixture-west", SSLMode: "disable"}}}
	open := func(_ context.Context, node database.Connection) (*sql.DB, error) {
		return stdlib.OpenDB(*configurations[node.Name]), nil
	}
	t.Run("source write fence", func(t *testing.T) {
		exercisePostgresSQLFences(t, settings, keys, open, writers)
	})
	report := inspectPlacement(t.Context(), settings, keys, readSnapshots(open), requireRecordedPin)
	if !report.Complete || len(report.Issues) != 0 || len(report.Schools) != 2 {
		t.Fatalf("native PostgreSQL read failed: %#v", report)
	}
	for _, pool := range report.Pools {
		if len(pool.Tables) != 45 {
			t.Fatal("PostgreSQL omitted registered table counts")
		}
	}
	source := inventorySource(t.Context(), settings, keys, SourceSelection{Version: 1, Schools: []string{"east", "west"}}, readSnapshots(open))
	if !source.Complete || len(source.Issues) != 0 || len(source.SHA256) != 64 || len(source.Pools) != 2 || len(source.Pools[0].Tables)+len(source.Pools[1].Tables) != 102 {
		t.Fatalf("registered PostgreSQL source inventory failed: %#v", source)
	}
	// Exercise the same transaction constructor used by every inspector source.
	reader, err := open(t.Context(), settings.Control)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	tx, err := beginInspectionSnapshot(t.Context(), reader, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	var readOnly string
	if err := tx.QueryRow("SHOW transaction_read_only").Scan(&readOnly); err != nil || readOnly != "on" {
		t.Fatalf("snapshot is not read-only: %v", err)
	}
	if _, err := tx.Exec("DELETE FROM identity_schools"); err == nil {
		t.Fatal("PostgreSQL inspection snapshot allowed mutation")
	}
	tx.Rollback()
	var identities int
	if err := control.QueryRow("SELECT count(*) FROM identity_schools").Scan(&identities); err != nil || identities != 2 {
		t.Fatal("inspection changed PostgreSQL source")
	}
	if _, err := west.Exec("INSERT INTO students_records VALUES ('orphan','orphan-record','Secret','Student','active',1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := control.Exec("UPDATE edu_tenant_placement SET pool_id='fixture-tenant_west' WHERE tenant_id='east'"); err != nil {
		t.Fatal(err)
	}
	if _, err := west.Exec("INSERT INTO edu_outbox(event_id,tenant_id,event_type,aggregate_id,payload,created_at,available_at,lease_id,lease_until) VALUES ('leased','west','students.registered.v1','same-record','{}',1,1,'lease',4102444800000)"); err != nil {
		t.Fatal(err)
	}
	damaged := inspectPlacement(t.Context(), settings, keys, readSnapshots(open), requireRecordedPin)
	codes := map[string]bool{}
	for _, issue := range damaged.Issues {
		codes[issue.Code] = true
	}
	for _, code := range []string{"unknown_scope", "wrong_pin_pool", "active_outbox_lease", "unexpected_pool_rows"} {
		if !codes[code] {
			t.Errorf("PostgreSQL missed %s", code)
		}
	}
}
