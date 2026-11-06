package edumigrations

import (
	"testing"

	"github.com/elcomware/edupilot/platform/database"
	"github.com/goravel/framework/contracts/database/schema"
)

func TestRegisterExpandsTenantPlansAndRestrictsControlPlans(t *testing.T) {
	config, err := database.ReadSettings(func(k, fallback string) string {
		if k == "EDUPILOT_DEDICATED_DATABASES" {
			return `[{"tenant":"west","driver":"sqlite","database":"west.sqlite"}]`
		}
		return fallback
	})
	if err != nil {
		t.Fatal(err)
	}
	p := Plan{ID: "20260102000000_auth", ControlOnly: true}
	all := Register(config, PlacementPlan(), OutboxPlan(), p)
	want := []string{
		"20260101000001_edu_placement__sqlite", "20260101000001_edu_placement__tenant_west",
		"20260101000002_edu_outbox__sqlite", "20260101000002_edu_outbox__tenant_west",
		"20260102000000_auth__sqlite",
	}
	if len(all) != len(want) {
		t.Fatalf("registered %d migrations, want %d", len(all), len(want))
	}
	for i, m := range all {
		if m.Signature() != want[i] {
			t.Errorf("unexpected migration %d: %s", i, m.Signature())
		}
		if _, switchesFrameworkConnection := m.(schema.Connection); switchesFrameworkConnection {
			t.Fatal("native migrator must stay on its control connection")
		}
		connection := m.(*registered).node.Name
		if connection != config.Nodes()[i%2].Name {
			t.Errorf("migration %s uses %q", m.Signature(), connection)
		}
	}
}
