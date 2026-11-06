package database

import (
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestSQLiteFileDSNOpensExactFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "school 100% & plus+.sqlite")
	db, err := sql.Open("sqlite3", SQLiteFileDSN(path, url.Values{"mode": {"rwc"}, "_pragma": {"foreign_keys(ON)"}}))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE demo(id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	var foreignKeys int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		t.Fatalf("foreign keys = %d, error = %v", foreignKeys, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}

	reader, err := sql.Open("sqlite3", SQLiteFileDSN(path, url.Values{"mode": {"ro"}}))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	var count int
	if err := reader.QueryRow("SELECT COUNT(*) FROM demo").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.Exec("INSERT INTO demo DEFAULT VALUES"); err == nil {
		t.Fatal("read-only connection admitted a write")
	}
}
