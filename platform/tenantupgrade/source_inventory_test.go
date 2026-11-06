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
	"reflect"
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

func validReconciledSourceSelection() string {
	return `{"version":2,"schools":["east","west"],"documents":{"version":2,"schools":2,"references":2,"documents":2,"contentBytes":301,"referencesSha256":"` + strings.Repeat("a", 64) + `","inventorySha256":"` + strings.Repeat("b", 64) + `","sha256":"92b406b1452cdbd7ce4a927ed31b05b7301a7921401c61f8be3abea8409728f9"}}`
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

func TestReadSourceSelectionVersion2RequiresReconciledObjectEvidence(t *testing.T) {
	selection, err := ReadSourceSelection(strings.NewReader(validReconciledSourceSelection()))
	if err != nil {
		t.Fatal(err)
	}
	if selection.Version != 2 || !selection.Documents.Present || selection.Documents.Version != 2 ||
		selection.Documents.References != 2 || selection.Documents.ReferencesSHA256 != strings.Repeat("a", 64) ||
		selection.Documents.InventorySHA256 != strings.Repeat("b", 64) || selection.Documents.SHA256 != "92b406b1452cdbd7ce4a927ed31b05b7301a7921401c61f8be3abea8409728f9" {
		t.Fatalf("reconciled evidence was not preserved: %#v", selection)
	}
	for name, raw := range map[string]string{
		"missing evidence":         `{"version":2,"schools":["east","west"]}`,
		"legacy evidence":          strings.Replace(validReconciledSourceSelection(), `"version":2,"schools":2,"references":2,"documents":2,"contentBytes":301,"referencesSha256":"`+strings.Repeat("a", 64)+`","inventorySha256":"`+strings.Repeat("b", 64)+`","sha256":"92b406b1452cdbd7ce4a927ed31b05b7301a7921401c61f8be3abea8409728f9"`, `"version":1,"schools":2,"documents":2,"contentBytes":301,"sha256":"`+strings.Repeat("c", 64)+`"`, 1),
		"missing reference digest": strings.Replace(validReconciledSourceSelection(), `,"referencesSha256":"`+strings.Repeat("a", 64)+`"`, ``, 1),
		"missing byte digest":      strings.Replace(validReconciledSourceSelection(), `,"inventorySha256":"`+strings.Repeat("b", 64)+`"`, ``, 1),
		"count mismatch":           strings.Replace(validReconciledSourceSelection(), `"references":2`, `"references":1`, 1),
		"aggregate mismatch":       strings.Replace(validReconciledSourceSelection(), `92b406b1452cdbd7ce4a927ed31b05b7301a7921401c61f8be3abea8409728f9`, strings.Repeat("c", 64), 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ReadSourceSelection(strings.NewReader(raw)); err == nil {
				t.Fatal("incomplete reconciled evidence accepted")
			}
		})
	}
}

func TestVersion2SourceDigestBindsReferencesAndVerifiedBytes(t *testing.T) {
	selection, err := ReadSourceSelection(strings.NewReader(validReconciledSourceSelection()))
	if err != nil {
		t.Fatal(err)
	}
	report := SourceInventory{
		Version: 2, Classification: "frozen-source-sql-objects", Complete: true, Schools: 2,
		Documents: &selection.Documents,
		Pools:     []SourcePoolInventory{{Name: "control", SHA256: strings.Repeat("d", 64)}},
	}
	first := digestSourceReport(report)
	changed := report
	documents := *report.Documents
	documents.ReferencesSHA256 = strings.Repeat("e", 64)
	changed.Documents = &documents
	if len(first) != 64 || digestSourceReport(changed) == first {
		t.Fatal("complete source digest did not bind the preserved reference set")
	}
	documents = *report.Documents
	documents.InventorySHA256 = strings.Repeat("f", 64)
	changed.Documents = &documents
	if digestSourceReport(changed) == first {
		t.Fatal("complete source digest did not bind the independently approved byte inventory")
	}
}

func TestSourceInventoryJSONPreservesVersion1ShapeAndMarksVersion2Ready(t *testing.T) {
	legacy, err := json.Marshal(SourceInventory{
		Version: 1, Classification: "frozen-source-sql", Complete: true,
		Schools: 2, Pools: []SourcePoolInventory{}, Issues: []SourceInventoryIssue{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(legacy, []byte(`"readyForCopy"`)) {
		t.Fatal("version 1 report shape changed")
	}
	reconciled, err := json.Marshal(SourceInventory{
		Version: 2, Classification: "frozen-source-sql-objects", Complete: true, ReadyForCopy: true,
		Schools: 2, Pools: []SourcePoolInventory{}, Issues: []SourceInventoryIssue{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(reconciled, []byte(`"readyForCopy":true`)) {
		t.Fatal("version 2 report did not expose copy readiness")
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

func TestPersistedSourceSelectionRoundTrips(t *testing.T) {
	empty := `{"version":2,"schools":["east"],"documents":{"version":2,"schools":1,"references":0,"documents":0,"contentBytes":0,"referencesSha256":"` + strings.Repeat("a", 64) + `","inventorySha256":"` + strings.Repeat("b", 64) + `","sha256":"09f8ce145f5872b07ab4b09270b3b64d4715b70904dbb58e7b5c199bb56892d0"}}`
	for name, original := range map[string]string{
		"SQL only":               `{"version":1,"schools":["east"]}`,
		"legacy documents":       validSourceSelection(),
		"reconciled documents":   validReconciledSourceSelection(),
		"no document references": empty,
	} {
		t.Run(name, func(t *testing.T) {
			want, err := ReadSourceSelection(strings.NewReader(original))
			if err != nil {
				t.Fatal(err)
			}
			stored, err := json.Marshal(want)
			if err != nil {
				t.Fatal(err)
			}
			got, err := ReadSourceSelection(bytes.NewReader(stored))
			if err != nil || !reflect.DeepEqual(want, got) {
				t.Fatal("saved selection could not reproduce the input evidence", err)
			}
		})
	}
}

func TestPersistedEmptyReferenceListRoundTrips(t *testing.T) {
	refs := ObjectReferences{Version: 1, Schools: []string{"east"}}
	expected, err := refs.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	stored, err := json.Marshal(refs)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadObjectReferences(bytes.NewReader(stored))
	if err != nil {
		t.Fatal("saved empty reference list is invalid", err)
	}
	digest, err := got.Fingerprint()
	if err != nil || digest != expected {
		t.Fatal("empty reference set digest changed during storage")
	}
}
