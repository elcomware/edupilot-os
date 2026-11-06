package tenantupgrade

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/platform/database"
)

func TestInspectionSQLiteIsReadOnlyAtHandleAndTransaction(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.sqlite")
	writer, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Exec("CREATE TABLE private_rows (school_id TEXT)"); err != nil {
		t.Fatal(err)
	}
	writer.Close()
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	source, err := openInspectionDB(t.Context(), database.Connection{Driver: "sqlite", Database: path})
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if _, err := source.Exec("INSERT INTO private_rows VALUES ('must-not-write')"); err == nil {
		t.Fatal("inspection handle permitted a write")
	}
	tx, err := beginInspectionSnapshot(t.Context(), source, "sqlite")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec("INSERT INTO private_rows VALUES ('must-not-write')"); err == nil {
		t.Fatal("inspection transaction permitted a write")
	}
	tx.Rollback()
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Fatal("read-only source changed")
	}
	missing := filepath.Join(t.TempDir(), "absent.sqlite")
	if db, err := openInspectionDB(t.Context(), database.Connection{Driver: "sqlite", Database: missing}); err == nil {
		db.Close()
		t.Fatal("opened missing source")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("missing source was created")
	}
}
func TestInspectionRejectsRegistryAndConfigurationWithoutLeaking(t *testing.T) {
	report := InspectPlacement(t.Context(), database.Settings{Control: database.Connection{Name: "secret-name", Driver: "postgres", Password: "private-password", Host: "private-host"}}, nil)
	if report.Complete || len(report.Issues) != 1 || report.Issues[0].Code != "invalid_configuration" {
		t.Fatal("invalid configuration accepted")
	}
	encoded, _ := json.Marshal(report)
	if strings.Contains(string(encoded), "private") || strings.Contains(string(encoded), "secret-name") {
		t.Fatal("configuration leaked")
	}
}
