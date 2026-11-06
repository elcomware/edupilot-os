package edumigrations

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/elcomware/edupilot/platform/database"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func transaction(t *testing.T, db *sql.DB, m *registered, down bool) error {
	t.Helper()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = m.apply(down, func(query string, args ...any) (int64, error) {
		result, err := tx.Exec(query, args...)
		if err != nil {
			return 0, err
		}
		return result.RowsAffected()
	})
	if err != nil {
		return err
	}
	return tx.Commit()
}

func TestRetryAfterTargetCommitWithoutControlLedgerUpdate(t *testing.T) {
	control, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "control.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer control.Close()
	dedicated, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "dedicated.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer dedicated.Close()
	m := &registered{node: database.Connection{Name: "tenant_west"}, plan: Plan{
		ID: "20260103000000_plain", Up: []string{"CREATE TABLE plain (id INTEGER)", "INSERT INTO plain VALUES (1)"},
		Down: []string{"DROP TABLE plain"},
	}}
	for i := 0; i < 2; i++ {
		if err := transaction(t, dedicated, m, false); err != nil {
			t.Fatalf("apply %d: %v", i, err)
		}
	}
	var count int
	if err := dedicated.QueryRow("SELECT count(*) FROM plain").Scan(&count); err != nil || count != 1 {
		t.Fatalf("unexpected number of inserted rows: %d %v", count, err)
	}
	if err := control.QueryRow("SELECT count(*) FROM sqlite_master WHERE name IN ('edu_migration_receipts','plain')").Scan(&count); err != nil || count != 0 {
		t.Fatalf("control leaked dedicated DDL: %d %v", count, err)
	}
	for i := 0; i < 2; i++ {
		if err := transaction(t, dedicated, m, true); err != nil {
			t.Fatalf("rollback %d: %v", i, err)
		}
	}
	if err := transaction(t, dedicated, m, false); err != nil {
		t.Fatalf("reapply after rollback: %v", err)
	}
}

func TestFailedDDLDoesNotLeaveAReceipt(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "ddl.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	m := &registered{node: database.Connection{Name: "sqlite"}, plan: Plan{
		ID: "20260104000000_broken", Up: []string{"CREATE TABLE demo (id INTEGER)", "THIS IS NOT SQL"},
	}}
	if err := transaction(t, db, m, false); err == nil {
		t.Fatal("broken statement committed")
	}
	m.plan.Up = m.plan.Up[:1]
	if err := transaction(t, db, m, false); err != nil {
		t.Fatalf("retry after repaired migration: %v", err)
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM edu_migration_receipts").Scan(&count); err != nil || count != 1 {
		t.Fatalf("missing applied receipt: %d %v", count, err)
	}
}
