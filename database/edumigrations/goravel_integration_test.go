package edumigrations_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/elcomware/edupilot-os/app/facades"
	"github.com/elcomware/edupilot-os/database/edumigrations"
	"github.com/elcomware/edupilot-os/platform/database"
	goraveldatabase "github.com/goravel/framework/database"
	"github.com/goravel/framework/database/migration"
	"github.com/goravel/framework/log"
	"github.com/goravel/sqlite"
)

// Exercise the actual v1.18 provider, schema transaction and migration repository
// across two SQLite files; no HTTP server or user database is involved.
func TestGoravelMigratorKeepsHistoryOnControlAndDDLOnItsNamedPool(t *testing.T) {
	dir := t.TempDir()
	settings := database.Settings{
		Control:   database.Connection{Name: "sqlite", Driver: "sqlite", Database: filepath.Join(dir, "control.sqlite")},
		Dedicated: []database.Connection{{Name: "tenant_west", Driver: "sqlite", Tenant: "west", Database: filepath.Join(dir, "west.sqlite")}},
	}
	app := facades.App()
	// Each native migrator specification owns its provider registry and SQL pools.
	app.Fresh()
	t.Cleanup(func() { app.Fresh() })
	facades.Config().Add("database", map[string]any{
		"default": "sqlite", "connections": settings.Connections(),
		"pool": map[string]any{"max_open_conns": 3, "max_idle_conns": 3},
	})
	(&log.ServiceProvider{}).Register(app)
	(&sqlite.ServiceProvider{}).Register(app)
	(&goraveldatabase.ServiceProvider{}).Register(app)
	schema := facades.Schema()
	if schema == nil {
		t.Fatal("missing Goravel schema")
	}
	controlPlan := edumigrations.Plan{ID: "20260102000001_control", ControlOnly: true,
		Up: []string{"CREATE TABLE auth_fixture (id INTEGER PRIMARY KEY)"}, Down: []string{"DROP TABLE auth_fixture"}}
	dataPlan := edumigrations.Plan{ID: "20260102000002_data",
		Up:   []string{"CREATE TABLE tenant_fixture (id INTEGER PRIMARY KEY)", "INSERT INTO tenant_fixture VALUES (9)"},
		Down: []string{"DROP TABLE tenant_fixture"}}
	schema.Register(edumigrations.Register(settings, edumigrations.PlacementPlan(), controlPlan, dataPlan))
	// Warm both named ORM connections before the native migrator runs, just as
	// the application service provider does. v1.18 caches the query handles.
	catalog, err := database.Open(settings, func(node database.Connection) (*sql.DB, error) {
		return facades.Orm().Connection(node.Name).DB()
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	migrator := migration.NewMigrator(nil, schema, "migrations")
	if err := migrator.Run(); err != nil {
		t.Fatal(err)
	}
	if err := edumigrations.CheckReady(ctx, catalog, schema.Migrations()); err != nil {
		t.Fatal(err)
	}
	for _, node := range settings.Nodes() {
		db, err := facades.Orm().Connection(node.Name).DB()
		if err != nil {
			t.Fatal(err)
		}
		p := database.Pool{DB: db, Driver: node.Driver}
		count(t, p, "SELECT count(*) FROM tenant_fixture", 1)
		if node.Tenant == "" {
			count(t, p, "SELECT count(*) FROM migrations", 5)
			count(t, p, "SELECT count(*) FROM sqlite_master WHERE name = 'auth_fixture'", 1)
		} else {
			count(t, p, "SELECT count(*) FROM sqlite_master WHERE name IN ('migrations', 'auth_fixture')", 0)
		}
	}
	central, err := facades.Orm().Connection("sqlite").DB()
	if err != nil {
		t.Fatal(err)
	}
	// Simulate the crash between target commit and control Log. The next Run
	// repairs the ledger with no second CREATE or INSERT on the dedicated DB.
	if _, err := central.Exec("DELETE FROM migrations WHERE migration = ?", dataPlan.ID+"__tenant_west"); err != nil {
		t.Fatal(err)
	}
	if err := edumigrations.CheckReady(ctx, catalog, schema.Migrations()); err == nil {
		t.Fatal("readiness accepted a missing control-ledger entry")
	}
	if err := migrator.Run(); err != nil {
		t.Fatal(err)
	}
	count(t, database.Pool{DB: central, Driver: "sqlite"}, "SELECT count(*) FROM migrations", 5)
	// A restored target can lose one receipt while all native ledger rows and
	// ownership markers survive. The deployment runner must still return an error.
	west := catalog.Named()["tenant_west"]
	id := dataPlan.ID + "__tenant_west"
	if _, err := west.DB.Exec("DELETE FROM edu_migration_receipts WHERE signature = ?", id); err != nil {
		t.Fatal(err)
	}
	if err := edumigrations.Run(ctx, catalog); err == nil {
		t.Fatal("migration runner accepted incomplete dedicated DB receipts")
	}
	if _, err := west.DB.Exec("INSERT INTO edu_migration_receipts (signature) VALUES (?)", id); err != nil {
		t.Fatal(err)
	}
	if err := migrator.Rollback(5, 0); err != nil {
		t.Fatal(err)
	}
	for _, node := range settings.Nodes() {
		db, err := facades.Orm().Connection(node.Name).DB()
		if err != nil {
			t.Fatal(err)
		}
		count(t, database.Pool{DB: db, Driver: node.Driver}, "SELECT count(*) FROM sqlite_master WHERE name IN ('tenant_fixture', 'auth_fixture', 'edu_pool_marker')", 0)
		db.Close()
	}
}
