package edumigrations_test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/elcomware/edupilot-os/platform/database"
)

type starterTableOwnership struct{ Name, Owner, Classification, Placement, ScopeColumn string }

func starterTableInventory(t *testing.T) []starterTableOwnership {
	t.Helper()
	data, err := os.ReadFile("testdata/starter/tables.json")
	if err != nil {
		t.Fatalf("exact starter table ownership required: %v", err)
	}
	var tables []starterTableOwnership
	if err := json.Unmarshal(data, &tables); err != nil {
		t.Fatal(err)
	}
	return tables
}
func TestStarterFixtureTableOwnershipManifest(t *testing.T) { starterTableInventory(t) }
func verifyStarterTables(t *testing.T, f StarterFixture) {
	t.Helper()
	inventory := starterTableInventory(t)
	data, err := os.ReadFile("../../docs/architecture/tenant-ownership-inventory.md")
	if err != nil {
		t.Fatal(err)
	}
	documented := map[string][]string{}
	for _, line := range strings.Split(string(data), "\n") {
		cells := strings.Split(line, "|")
		if len(cells) == 8 {
			for i := range cells {
				cells[i] = strings.TrimSpace(cells[i])
			}
			documented[cells[1]] = cells[2:7]
		}
	}
	for _, table := range inventory {
		row := documented[table.Name]
		if len(row) == 0 || row[0] != table.Owner || row[1] != table.Classification {
			t.Fatalf("table ownership annotation mismatch for %s", table.Name)
		}
	}
	for _, pool := range f.Catalog.All() {
		actual, err := starterActualTables(pool)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateStarterTableNames(inventory, pool.DB == f.Control.DB, actual); err != nil {
			t.Fatalf("%s: %v", poolName(f, pool), err)
		}
		for _, table := range inventory {
			if table.ScopeColumn != "" && (table.Placement == "all" || pool.DB == f.Control.DB) {
				// Reading a declared scope column must succeed on the actual durable table.
				if _, err := pool.DB.Exec("SELECT " + table.ScopeColumn + " FROM " + table.Name + " WHERE 1=0"); err != nil {
					t.Fatalf("owner column for %s missing: %v", table.Name, err)
				}
			}
		}
	}
}
func starterActualTables(pool database.Pool) ([]string, error) {
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"
	if pool.Driver == "postgres" {
		query = "SELECT table_name FROM information_schema.tables WHERE table_schema=current_schema() AND table_type='BASE TABLE' ORDER BY table_name"
	}
	rows, err := pool.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}
func TestStarterFixtureExactTablesRejectsUnexpectedTable(t *testing.T) {
	f := starterFixture(t, "shared")
	tables, err := starterActualTables(f.Control)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Control.DB.Exec("CREATE TABLE unexpected_private (school_id TEXT)"); err != nil {
		t.Fatal(err)
	}
	changed, err := starterActualTables(f.Control)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(tables, changed) {
		t.Fatal("unexpected durable table was not discovered")
	}
	if err := validateStarterTableNames(starterTableInventory(t), true, changed); err == nil {
		t.Fatal("unexpected durable table accepted by ownership check")
	}
	if len(changed) != len(starterTableInventory(t))+1 {
		t.Fatal(fmt.Sprint("unexpected table baseline: ", changed))
	}
}

func validateStarterTableNames(inventory []starterTableOwnership, control bool, actual []string) error {
	expected := []string{}
	for _, table := range inventory {
		if table.Placement != "all" && table.Placement != "control" {
			return fmt.Errorf("unknown placement annotation %s", table.Name)
		}
		if table.Placement == "all" || control {
			expected = append(expected, table.Name)
		}
	}
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		return fmt.Errorf("exact durable table set differs: got %v; want %v", actual, expected)
	}
	return nil
}
