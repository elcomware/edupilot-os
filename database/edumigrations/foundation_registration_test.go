package edumigrations_test

import (
	"os"
	"strings"
	"testing"

	"github.com/elcomware/edupilot-os/app/facades"
	"github.com/elcomware/edupilot-os/bootstrap"
	"github.com/elcomware/edupilot-os/database/edumigrations"
	"github.com/elcomware/edupilot-os/platform/database"
	schemaContract "github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/database/migration"
)

// Upgrade an actual frozen starter and deliberately lose the control ledger
// entry for committed dedicated DDL. Replaying must preserve both receipts and
// records. This test never activates a tenant or copies an identity.
func TestFoundationDDLAllPoolsAndReplay(t *testing.T) {
	for _, driver := range []string{"sqlite", "postgres"} {
		t.Run(driver, func(t *testing.T) {
			if driver == "postgres" && os.Getenv("EDUPILOT_TEST_POSTGRES_DSN") == "" {
				t.Skip("requires disposable EDUPILOT_TEST_POSTGRES_DSN")
			}
			f := starterFixtureDriver(t, "mixed", driver)
			schema := facades.Schema()
			var additions []schemaContract.Migration
			for _, m := range bootstrap.Migrations() {
				if strings.HasPrefix(m.Signature(), "202611060100") {
					additions = append(additions, m)
				}
			}
			if len(additions) != 94 {
				t.Fatalf("expected 48 control/shared and 46 dedicated registrations, got %d", len(additions))
			}
			schema.Register(additions)
			migrator := migration.NewMigrator(nil, schema, "migrations")
			if err := migrator.Run(); err != nil {
				t.Fatal(err)
			}
			if err := edumigrations.CheckReady(t.Context(), f.Catalog, schema.Migrations()); err != nil {
				t.Fatal(err)
			}
			for _, n := range []string{"tenants", "database_pools", "identity_sessions_v2", "identity_credentials_v2"} {
				checkFoundationTable(t, f.Control, n, 1)
				for _, pool := range f.Dedicated {
					checkFoundationTable(t, pool, n, 0)
				}
			}
			for _, pool := range f.Catalog.All() {
				for _, name := range []string{"tenant_gates", "tenant_upgrade_receipts", "tenancy_schools", "students_records_v2", "ws_attendance_v2", "documents_metadata_v2", "edu_outbox_v2"} {
					checkFoundationTable(t, pool, name, 1)
				}
			}
			var signature string
			for _, m := range additions {
				if strings.HasPrefix(m.Signature(), "20261106010003_topology__tenant_") {
					signature = m.Signature()
					break
				}
			}
			if signature == "" {
				t.Fatal("missing dedicated topology plan")
			}
			if _, err := f.Control.DB.Exec(f.Control.Bind("DELETE FROM migrations WHERE migration=?"), signature); err != nil {
				t.Fatal(err)
			}
			if err := edumigrations.CheckReady(t.Context(), f.Catalog, schema.Migrations()); err == nil {
				t.Fatal("missing control ledger went undetected")
			}
			if err := migrator.Run(); err != nil {
				t.Fatal(err)
			}
			if err := edumigrations.CheckReady(t.Context(), f.Catalog, schema.Migrations()); err != nil {
				t.Fatal(err)
			}
			// Forward-only tables must survive an operator rollback attempt.
			if err := migrator.Rollback(1, 0); err == nil {
				t.Fatal("rollback accepted a protected foundation migration")
			}
			if err := edumigrations.CheckReady(t.Context(), f.Catalog, schema.Migrations()); err != nil {
				t.Fatal(err)
			}
			// All tenant gates and copied records must still be empty before explicit upgrade.
			for _, pool := range f.Catalog.All() {
				count(t, pool, "SELECT count(*) FROM tenant_gates", 0)
				count(t, pool, "SELECT count(*) FROM students_records_v2", 0)
			}
		})
	}
}
func checkFoundationTable(t *testing.T, p database.Pool, name string, want int) {
	t.Helper()
	q := "SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?"
	if p.Driver == "postgres" {
		q = "SELECT count(*) FROM information_schema.tables WHERE table_schema=current_schema() AND table_name=?"
	}
	var got int
	if err := p.DB.QueryRow(p.Bind(q), name).Scan(&got); err != nil || got != want {
		t.Fatalf("table %s: got %d want %d err %v", name, got, want, err)
	}
}
