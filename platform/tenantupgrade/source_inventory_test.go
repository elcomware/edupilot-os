package tenantupgrade

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func validSourceSelection() string {
	return `{"version":1,"schools":["east","west"],"documents":{"version":1,"schools":2,"documents":7,"contentBytes":301,"sha256":"` + strings.Repeat("a", 64) + `"}}`
}

func TestReadSourceSelectionAcceptsOnlyLegacySourceIdentity(t *testing.T) {
	selection, err := ReadSourceSelection(strings.NewReader(validSourceSelection()))
	if err != nil {
		t.Fatal(err)
	}
	if !selection.Documents.Present || selection.Documents.SHA256 != strings.Repeat("a", 64) ||
		len(selection.Schools) != 2 || selection.Schools[0] != "east" || selection.Schools[1] != "west" {
		t.Fatalf("unexpected source selection: %#v", selection)
	}
	selection, err = ReadSourceSelection(strings.NewReader(`{"version":1,"schools":["east","west"]}`))
	if err != nil || selection.Documents.Present {
		t.Fatalf("SQL-only source selection was not accepted: %#v, %v", selection, err)
	}

	for name, raw := range map[string]string{
		"target identity":    strings.Replace(validSourceSelection(), `"schools":[`, `"tenant":"00000000-0000-4000-8000-000000000001","schools":[`, 1),
		"duplicate school":   strings.Replace(validSourceSelection(), `"east","west"`, `"east","east"`, 1),
		"unsorted schools":   strings.Replace(validSourceSelection(), `"east","west"`, `"west","east"`, 1),
		"bad document count": strings.Replace(validSourceSelection(), `"schools":2,"documents"`, `"schools":1,"documents"`, 1),
		"trailing input":     validSourceSelection() + `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ReadSourceSelection(strings.NewReader(raw)); err == nil {
				t.Fatal("invalid source selection accepted")
			}
		})
	}
}

func TestSourceTableInventoryIsLogicalOrderedAndContentSensitive(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`CREATE TABLE fixture (
		school_id TEXT NOT NULL, id TEXT NOT NULL, note TEXT, version BIGINT NOT NULL,
		payload BLOB NOT NULL, occurred_at TIMESTAMP NOT NULL, PRIMARY KEY (school_id,id))`)
	if err != nil {
		t.Fatal(err)
	}
	spec := sourceTable{
		name: "fixture", scope: "school_id", key: []string{"school_id", "id"},
		columns: []sourceColumn{
			{"school_id", sourceText, false}, {"id", sourceText, false},
			{"note", sourceText, true}, {"version", sourceInteger, false},
			{"payload", sourceBinary, false}, {"occurred_at", sourceTimestamp, false},
		},
	}
	insert := func(school, id string, note any, version int64, payload []byte, occurred string) {
		t.Helper()
		if _, err := db.Exec("INSERT INTO fixture VALUES (?,?,?,?,?,?)", school, id, note, version, payload, occurred); err != nil {
			t.Fatal(err)
		}
	}
	insert("west", "same-id", nil, 2, []byte{0, 1, 0xff}, "1985-01-02T03:04:05Z")
	insert("east", "same-id", "élève", 1, []byte{}, "1985-01-02T03:04:05.000000000Z")
	first := inventoryTestTable(t, db, spec)
	if first.Rows != 2 || len(first.SHA256) != 64 || len(first.SchemaSHA256) != 64 {
		t.Fatalf("incomplete inventory: %#v", first)
	}
	if _, err := db.Exec("DELETE FROM fixture"); err != nil {
		t.Fatal(err)
	}
	insert("east", "same-id", "élève", 1, []byte{}, "1985-01-02 03:04:05+00:00")
	insert("west", "same-id", nil, 2, []byte{0, 1, 0xff}, "1985-01-02T03:04:05Z")
	second := inventoryTestTable(t, db, spec)
	if first != second {
		t.Fatalf("physical insertion order or equivalent timestamp changed inventory:\n%#v\n%#v", first, second)
	}
	if _, err := db.Exec("UPDATE fixture SET note='élèvé' WHERE school_id='east'"); err != nil {
		t.Fatal(err)
	}
	changed := inventoryTestTable(t, db, spec)
	if changed.SHA256 == first.SHA256 {
		t.Fatal("private row content change did not change digest")
	}
	encoded, err := json.Marshal(changed)
	if err != nil || bytes.Contains(encoded, []byte("élèvé")) || bytes.Contains(encoded, []byte("same-id")) {
		t.Fatal("inventory report disclosed private row values")
	}
}

func TestSourceTableInventoryRejectsIncompatibleSchemaAndUnknownSchool(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec("CREATE TABLE fixture (school_id TEXT NOT NULL, id TEXT NOT NULL, version TEXT NOT NULL, PRIMARY KEY (school_id,id))"); err != nil {
		t.Fatal(err)
	}
	spec := sourceTable{name: "fixture", scope: "school_id", key: []string{"school_id", "id"}, columns: []sourceColumn{
		{"school_id", sourceText, false}, {"id", sourceText, false}, {"version", sourceInteger, false},
	}}
	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, _, err := inventorySourceTable(t.Context(), tx, "sqlite", spec, map[string]bool{"east": true}); !errors.Is(err, ErrSourceSchema) {
		t.Fatalf("wrong column type accepted: %v", err)
	}
	_ = tx.Rollback()

	if _, err := db.Exec("DROP TABLE fixture"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE fixture (school_id TEXT NOT NULL, id TEXT NOT NULL, version BIGINT NOT NULL, PRIMARY KEY (school_id,id))"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO fixture VALUES ('orphan','private-id',1)"); err != nil {
		t.Fatal(err)
	}
	tx, err = db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	_, unknown, err := inventorySourceTable(context.Background(), tx, "sqlite", spec, map[string]bool{"east": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(unknown) != 1 || unknown[0] != "orphan" {
		t.Fatalf("unknown school not reported: %#v", unknown)
	}
}

func inventoryTestTable(t *testing.T, db *sql.DB, spec sourceTable) SourceTableInventory {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	report, unknown, err := inventorySourceTable(ctx, tx, "sqlite", spec, map[string]bool{"east": true, "west": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(unknown) != 0 {
		t.Fatalf("unexpected school scopes: %#v", unknown)
	}
	return report
}

func TestSourceTableInventoryMatchesPostgresAndSQLite(t *testing.T) {
	dsn := os.Getenv("EDUPILOT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set EDUPILOT_TEST_POSTGRES_DSN for portable source digest qualification")
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid disposable PostgreSQL test configuration")
	}
	admin := stdlib.OpenDB(*config)
	t.Cleanup(func() { admin.Close() })
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	schema := "edu_source_" + hex.EncodeToString(random[:])
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE") })
	config.RuntimeParams["search_path"] = schema
	postgres := stdlib.OpenDB(*config)
	t.Cleanup(func() { postgres.Close() })
	if _, err := postgres.Exec(`CREATE TABLE fixture (
		school_id TEXT NOT NULL, id TEXT NOT NULL, note TEXT, version BIGINT NOT NULL,
		payload BYTEA NOT NULL, occurred_at TIMESTAMPTZ NOT NULL, PRIMARY KEY (school_id,id))`); err != nil {
		t.Fatal(err)
	}
	sqlite, err := sql.Open("sqlite3", "file:portable-source?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlite.Close() })
	if _, err := sqlite.Exec(`CREATE TABLE fixture (
		school_id TEXT NOT NULL, id TEXT NOT NULL, note TEXT, version BIGINT NOT NULL,
		payload BLOB NOT NULL, occurred_at TIMESTAMP NOT NULL, PRIMARY KEY (school_id,id))`); err != nil {
		t.Fatal(err)
	}
	values := [][]any{
		{"east", "same-id", "élève", int64(1), []byte{}, "1985-01-02T03:04:05Z"},
		{"west", "same-id", nil, int64(2), []byte{0, 1, 0xff}, "1985-01-02T03:04:05Z"},
	}
	for _, row := range values {
		if _, err := sqlite.Exec("INSERT INTO fixture VALUES (?,?,?,?,?,?)", row...); err != nil {
			t.Fatal(err)
		}
		if _, err := postgres.Exec("INSERT INTO fixture VALUES ($1,$2,$3,$4,$5,$6)", row...); err != nil {
			t.Fatal(err)
		}
	}
	spec := sourceTable{name: "fixture", scope: "school_id", key: []string{"school_id", "id"}, columns: []sourceColumn{
		{"school_id", sourceText, false}, {"id", sourceText, false}, {"note", sourceText, true},
		{"version", sourceInteger, false}, {"payload", sourceBinary, false}, {"occurred_at", sourceTimestamp, false},
	}}
	sqliteReport := inventoryTestTable(t, sqlite, spec)
	tx, err := beginInspectionSnapshot(t.Context(), postgres, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	postgresReport, unknown, err := inventorySourceTable(t.Context(), tx, "postgres", spec, map[string]bool{"east": true, "west": true})
	if err != nil || len(unknown) != 0 {
		t.Fatalf("PostgreSQL source inventory failed: %#v %v", unknown, err)
	}
	if postgresReport != sqliteReport {
		t.Fatalf("portable digests differ:\nSQLite %#v\nPostgreSQL %#v", sqliteReport, postgresReport)
	}
}
