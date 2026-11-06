package edumigrations_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

type ownedDatabase struct {
	name string
	db   *sql.DB
}

type postgresDatabaseDrill struct {
	ctx            context.Context
	cfg            *pgx.ConnConfig
	sslmode        string
	tlsEnvironment map[string]string
	create         func(string) ownedDatabase
}

func (d postgresDatabaseDrill) settings(shared, west ownedDatabase) database.Settings {
	control := database.Connection{Name: "postgres", Driver: "postgres", Host: d.cfg.Host, Port: int(d.cfg.Port), Username: d.cfg.User, Password: d.cfg.Password, SSLMode: d.sslmode, Database: shared.name}
	dedicated := control
	dedicated.Name, dedicated.Tenant, dedicated.Database = "tenant_west", "west", west.name
	return database.Settings{Control: control, Dedicated: []database.Connection{dedicated}}
}

// Separate opt-in: an ordinary schema-scoped test DSN never authorizes creating
// databases. Public tables are populated only after this test creates a fresh,
// random database and verifies the actual opened handle's database identity.
func newPostgresDatabaseDrill(t *testing.T) postgresDatabaseDrill {
	t.Helper()
	if os.Getenv("EDUPILOT_TEST_POSTGRES_CREATE_DATABASES") != "1" || os.Getenv("EDUPILOT_TEST_POSTGRES_DSN") == "" {
		t.Skip("requires EDUPILOT_TEST_POSTGRES_DSN and EDUPILOT_TEST_POSTGRES_CREATE_DATABASES=1; native PostgreSQL CLI acceptance remains open")
	}
	cfg, sslmode, tlsEnvironment, err := inspectionCLIConfig(os.Getenv("EDUPILOT_TEST_POSTGRES_DSN"))
	if err != nil {
		t.Fatal(err) // Configuration errors intentionally contain no input or DSN.
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	t.Cleanup(cancel)
	admin := stdlib.OpenDB(*cfg)
	admin.SetMaxOpenConns(1)
	if err := admin.PingContext(ctx); err != nil {
		admin.Close()
		t.Fatal("cannot connect to PostgreSQL fixture administration database")
	}
	owned := []ownedDatabase{}
	t.Cleanup(func() {
		// Test contexts are cancelled before cleanup. Use a separate bounded
		// cleanup context and only names whose CREATE DATABASE succeeded below.
		cleanup, finish := context.WithTimeout(context.Background(), 20*time.Second)
		defer finish()
		defer admin.Close()
		for _, entry := range owned {
			if entry.db != nil {
				entry.db.Close()
			}
		}
		for _, entry := range owned {
			if _, err := admin.ExecContext(cleanup, "DROP DATABASE "+pgx.Identifier{entry.name}.Sanitize()); err != nil {
				t.Error("could not drop a test-created PostgreSQL fixture database")
			}
		}
	})
	create := func(label string) ownedDatabase {
		t.Helper()
		name := "edu_drill_" + label + "_" + inspectionCLIRandom(t)
		if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()+" TEMPLATE template0"); err != nil {
			t.Fatal("could not create a private PostgreSQL fixture database")
		}
		owned = append(owned, ownedDatabase{name: name})
		copy := cfg.Copy() // Preserve configured TLS, credentials and timeouts.
		copy.Database = name
		copy.RuntimeParams["search_path"] = "public"
		writer := stdlib.OpenDB(*copy)
		writer.SetMaxOpenConns(1)
		owned[len(owned)-1].db = writer
		var actual, schema string
		if err := writer.QueryRowContext(ctx, "SELECT current_database(),current_schema()").Scan(&actual, &schema); err != nil || actual != name || schema != "public" {
			t.Fatal("fixture handle does not identify its newly created database and public schema")
		}
		return owned[len(owned)-1]
	}
	return postgresDatabaseDrill{ctx, cfg, sslmode, tlsEnvironment, create}
}

// Only a database actually created by this test can supply the owning handle.
// Compare both connections before allowing Goravel to issue any DDL.
func (d ownedDatabase) beforeDDL(ctx context.Context, handle *sql.DB, apply func() error) error {
	if d.name == "" || d.db == nil || handle == nil {
		return errors.New("missing owned PostgreSQL database")
	}
	for _, connection := range []*sql.DB{d.db, handle} {
		name, schema, err := postgresHandleIdentity(connection)(ctx)
		if err != nil || name != d.name || schema != "public" {
			return errors.New("native connection does not select the owned database and public schema")
		}
	}
	return apply()
}
