package architecture

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/modules"
	authmigrations "github.com/elcomware/edupilot/modules/auth/database/migrations"
	enrollment "github.com/elcomware/edupilot/modules/enrollment/providers"
	students "github.com/elcomware/edupilot/modules/students/database/migrations"
	"github.com/elcomware/edupilot/platform/database"
)

// Changing a registered schema or adding durable storage without an ownership
// decision must fail this check. The document itself is the inventory, not a
// second hand-maintained list of expected business tables in the test.
func TestTenantOwnershipInventoryCoversRegisteredTables(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(t), "docs/architecture/tenant-ownership-inventory.md"))
	if err != nil {
		t.Fatalf("ownership inventory is required: %v", err)
	}
	foundation, err := os.ReadFile(filepath.Join(root(t), "docs/architecture/foundation-ownership-inventory.md"))
	if err != nil {
		t.Fatal(err)
	}
	data = append(append(data, '\n'), foundation...)
	entries := map[string][]string{}
	for _, line := range strings.Split(string(data), "\n") {
		cells := strings.Split(line, "|")
		if len(cells) != 8 || strings.TrimSpace(cells[1]) == "Surface" || strings.Contains(cells[1], "---") {
			continue
		}
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		name := strings.Trim(cells[1], "`")
		if _, found := entries[name]; found {
			t.Errorf("duplicate inventory surface %s", name)
		}
		entries[name] = cells[2:7]
		for _, field := range entries[name] {
			if field == "" {
				t.Errorf("%s has an unnamed ownership decision", name)
			}
		}
		if _, err := os.Stat(filepath.Join(root(t), cells[6])); err != nil {
			t.Errorf("%s source %s: %v", name, cells[6], err)
		}
	}
	require := func(name string) {
		t.Helper()
		if _, ok := entries[name]; !ok {
			t.Errorf("missing ownership/upgrade entry for %s", name)
		}
	}
	plans := []edumigrations.Plan{edumigrations.PlacementPlan(), edumigrations.OutboxPlan(), {ID: "identity", Up: authmigrations.SchemaUp}, {ID: "students", Up: students.SchemaUp}, {ID: "placements", Up: enrollment.SchemaUp}}
	definitions := modules.Definitions()
	registry, err := modules.NewRegistry(nil, definitions)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 42 {
		t.Fatalf("review module-count decision: got %d registrations, want 42", len(definitions))
	}
	workspaces := 0
	for _, d := range registry.Definitions {
		if d.Migration != nil {
			plans = append(plans, d.Migration())
			workspaces++
			require("ws_" + d.Key)
			require("ws_" + d.Key + "_v2")
			if row := entries["ws_"+d.Key]; len(row) > 0 && row[0] != "modules/"+d.Key {
				t.Errorf("workspace owner for %s = %s", d.Key, row[0])
			}
		} else if d.Key != "students" {
			t.Errorf("unexpected module without migration: %s", d.Key)
		}
	}
	if workspaces != 41 {
		t.Errorf("got %d work-list schemas, want 41; Students is specialized", workspaces)
	}
	table := regexp.MustCompile(`(?i)CREATE TABLE(?: IF NOT EXISTS)?\s+([a-z_][a-z0-9_]*)`)
	for _, p := range plans {
		for _, stmt := range p.Statements(database.Connection{Name: "sqlite", Driver: "sqlite"}, false) {
			for _, match := range table.FindAllStringSubmatch(stmt, -1) {
				require(match[1])
			}
		}
	}
	// Discover future literal SQL and native Blueprint tables outside registered
	// module plans as well. This catches an unregistered durable SQL surface.
	err = filepath.WalkDir(root(t), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".superpowers" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range table.FindAllStringSubmatch(string(source), -1) {
			require(match[1])
		}
		native := regexp.MustCompile(`\.Create\("([a-z_][a-z0-9_]*)"`)
		for _, match := range native.FindAllStringSubmatch(string(source), -1) {
			require(match[1])
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"migrations", "edu_migration_receipts", "jobs", "failed_jobs", "legacy-document-v1", "students-lifecycle-v1", "enrollment-events-v1", "workspace-events-v1", "legacy-session-token", "client-school-preferences"} {
		require(name)
	}
	for _, name := range []string{"jobs", "failed_jobs", "migrations", "edu_migration_receipts", "edu_pool_marker", "edu_tenant_placement"} {
		if row := entries[name]; len(row) > 0 && row[1] != "infrastructure/control" {
			t.Errorf("%s needs explicit infrastructure/control rationale", name)
		}
	}
}
