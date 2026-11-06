package edumigrations_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/elcomware/edupilot-os/modules"
	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/elcomware/edupilot-os/platform/tenantupgrade"
)

func TestStarterFixtureSourceInventoryIsCompleteAndPrivate(t *testing.T) {
	assertStarterFixtureSourceInventory(t, starterFixture(t, "mixed"))
}

func TestStarterFixturePostgresSourceInventory(t *testing.T) {
	if os.Getenv("EDUPILOT_TEST_POSTGRES_DSN") == "" {
		t.Skip("set EDUPILOT_TEST_POSTGRES_DSN for PostgreSQL source inventory")
	}
	assertStarterFixtureSourceInventory(t, starterFixtureDriver(t, "mixed", "postgres"))
}

func assertStarterFixtureSourceInventory(t *testing.T, fixture StarterFixture) {
	t.Helper()
	settings, err := database.ConfiguredSettings()
	if err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, 41)
	for _, definition := range modules.Definitions() {
		if definition.Migration != nil {
			keys = append(keys, definition.Key)
		}
	}
	schools := []string{fixture.Schools[0].School, fixture.Schools[1].School}
	if schools[1] < schools[0] {
		schools[0], schools[1] = schools[1], schools[0]
	}
	report := tenantupgrade.InventorySource(t.Context(), settings, keys, tenantupgrade.SourceSelection{Version: 1, Schools: schools})
	if !report.Complete || len(report.Issues) != 0 || len(report.SHA256) != 64 || len(report.Pools) != 2 {
		t.Fatalf("incomplete source inventory: %#v", report)
	}
	if len(report.Pools[0].Tables)+len(report.Pools[1].Tables) != 102 {
		t.Fatalf("expected 55 control and 47 dedicated inventories: %#v", report.Pools)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, school := range schools {
		if strings.Contains(string(encoded), `"`+school+`"`) {
			t.Fatal("source inventory disclosed a historical school identifier")
		}
	}
}
