package edumigrations_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"testing"
)

type starterPostgresTarget struct{ Database, Schema string }
type starterIdentityReader func(context.Context) (string, string, error)

func starterBeforePostgresDDL(ctx context.Context, expected starterPostgresTarget, identity starterIdentityReader, migrate func() error) error {
	if !regexp.MustCompile(`^edu_platform_[0-9a-f]{16}$`).MatchString(expected.Schema) || expected.Database == "" {
		return fmt.Errorf("target is not an owned disposable PostgreSQL schema")
	}
	actualDatabase, actualSchema, err := identity(ctx)
	if err != nil {
		return fmt.Errorf("verify actual Goravel PostgreSQL handle: %w", err)
	}
	if actualDatabase != expected.Database || actualSchema != expected.Schema {
		return fmt.Errorf("actual Goravel PostgreSQL handle differs from owned target")
	}
	return migrate()
}

func TestStarterFixtureRejectsWrongPostgresHandleBeforeDDL(t *testing.T) {
	for _, actual := range []starterPostgresTarget{{Database: "wrong", Schema: "edu_platform_0123456789abcdef"}, {Database: "expected", Schema: "public"}, {Database: "expected", Schema: "edu_platform_fedcba9876543210"}} {
		expected := starterPostgresTarget{Database: "expected", Schema: "edu_platform_0123456789abcdef"}
		db := sqlitePool(t, t.TempDir()+"/sentinel.sqlite")
		err := starterBeforePostgresDDL(t.Context(), expected, func(context.Context) (string, string, error) { return actual.Database, actual.Schema, nil }, func() error { _, err := db.Exec("CREATE TABLE forbidden_write(id INTEGER)"); return err })
		if err == nil {
			t.Errorf("wrong PostgreSQL handle allowed migration: %#v", actual)
		}
		var tables int
		if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name='forbidden_write'").Scan(&tables); err != nil {
			t.Fatal(err)
		}
		if tables != 0 {
			t.Fatal("DDL ran before target verification")
		}
	}
}
func postgresHandleIdentity(db *sql.DB) starterIdentityReader {
	return func(ctx context.Context) (string, string, error) {
		var database, schema string
		err := db.QueryRowContext(ctx, "SELECT current_database(),current_schema()").Scan(&database, &schema)
		return database, schema, err
	}
}

func TestStarterFixturePostgresWrongNativeHandle(t *testing.T) {
	if os.Getenv("EDUPILOT_TEST_POSTGRES_DSN") == "" {
		t.Skip("EDUPILOT_TEST_POSTGRES_DSN unavailable; live wrong-handle safety verification remains open")
	}
	fixture := starterFixtureDriver(t, "mixed", "postgres")
	first := fixture.Control.DB
	other := fixture.Dedicated["west"].DB
	database, schema, err := postgresHandleIdentity(first)(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	err = starterBeforePostgresDDL(t.Context(), starterPostgresTarget{Database: database, Schema: schema}, postgresHandleIdentity(other), func() error { _, err := other.Exec("CREATE TABLE forbidden_write(id INTEGER)"); return err })
	if err == nil {
		t.Fatal("wrong physical PostgreSQL handle accepted")
	}
	var count int
	if err := other.QueryRow("SELECT count(*) FROM information_schema.tables WHERE table_schema=current_schema() AND table_name='forbidden_write'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("DDL ran before rejecting wrong handle")
	}
}
