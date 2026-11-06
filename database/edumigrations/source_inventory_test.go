package edumigrations_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/modules"
	"github.com/elcomware/edupilot/modules/documents/adapters/outbound/blob"
	"github.com/elcomware/edupilot/platform/database"
	"github.com/elcomware/edupilot/platform/tenantupgrade"
)

func TestStarterFixtureSourceInventoryIsCompleteAndPrivate(t *testing.T) {
	assertStarterFixtureSourceInventory(t, starterFixture(t, "mixed"))
}

func TestStarterFixtureReconciledSourceInventoryIsReadyForCopy(t *testing.T) {
	fixture := starterFixture(t, "mixed")
	settings, err := database.ConfiguredSettings()
	if err != nil {
		t.Fatal(err)
	}
	mappings := make([]tenantupgrade.Mapping, 0, len(fixture.Schools))
	for i, school := range fixture.Schools {
		poolName := settings.Control.Name
		pool := fixture.Control
		if dedicated, ok := fixture.Dedicated[school.School]; ok {
			pool = dedicated
			poolName = "tenant_" + school.School
		}
		var marker string
		if err := pool.DB.QueryRow("SELECT pool_id FROM edu_pool_marker WHERE singleton=1").Scan(&marker); err != nil {
			t.Fatal(err)
		}
		id := func(n int) string { return fmt.Sprintf("%08x-0000-4000-8000-%012x", i+1, n+1) }
		mappings = append(mappings, tenantupgrade.Mapping{
			LegacySchool: school.School, SourcePool: poolName, SourceMarker: marker,
			Tenant: id(0), Organization: id(1), School: id(2), Campus: id(3), Timezone: "UTC", Locale: "en",
		})
	}
	slices.SortFunc(mappings, func(a, b tenantupgrade.Mapping) int { return strings.Compare(a.LegacySchool, b.LegacySchool) })
	manifest := tenantupgrade.Manifest{Version: 1, SourceSHA256: strings.Repeat("a", 64), Schools: mappings}
	references := tenantupgrade.ObjectReferences{Version: 1}
	for _, mapping := range mappings {
		references.Schools = append(references.Schools, mapping.LegacySchool)
	}
	for _, document := range fixture.Blobs {
		references.Objects = append(references.Objects, tenantupgrade.ObjectReference{School: document.School, ID: document.ID})
	}
	slices.SortFunc(references.Objects, func(a, b tenantupgrade.ObjectReference) int {
		if order := strings.Compare(a.School, b.School); order != 0 {
			return order
		}
		return strings.Compare(a.ID, b.ID)
	})
	baseline, err := blob.InventoryLocal(t.Context(), fixture.BlobRoot, manifest, "")
	if err != nil {
		t.Fatal(err)
	}
	reconciled, err := blob.ReconcileLocal(t.Context(), fixture.BlobRoot, manifest, references, baseline.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, 41)
	for _, definition := range modules.Definitions() {
		if definition.Migration != nil {
			keys = append(keys, definition.Key)
		}
	}
	selection := tenantupgrade.SourceSelection{Version: 2, Schools: references.Schools, Documents: tenantupgrade.SourceDocumentEvidence{
		Present: true, Version: reconciled.Version, Schools: reconciled.Schools, References: int64(reconciled.References),
		Documents: int64(reconciled.Documents), ContentBytes: reconciled.ContentBytes, ReferencesSHA256: reconciled.ReferencesSHA256,
		InventorySHA256: reconciled.InventorySHA256, SHA256: reconciled.SHA256,
	}}
	report := tenantupgrade.InventorySource(t.Context(), settings, keys, selection)
	if !report.Complete || !report.ReadyForCopy || report.Version != 2 || report.Classification != "frozen-source-sql-objects" || len(report.Issues) != 0 || len(report.SHA256) != 64 {
		t.Fatalf("reconciled source was not copy-ready: %#v", report)
	}
	legacy := tenantupgrade.InventorySource(t.Context(), settings, keys, tenantupgrade.SourceSelection{Version: 1, Schools: references.Schools})
	if !legacy.Complete || legacy.ReadyForCopy || legacy.SHA256 == report.SHA256 {
		t.Fatal("SQL-only preflight was treated as authoritative copy evidence")
	}
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
