package edumigrations_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/elcomware/edupilot-os/modules"
	"github.com/elcomware/edupilot-os/modules/documents/adapters/outbound/blob"
	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/elcomware/edupilot-os/platform/tenantupgrade"
)

func TestLegacyUpgradeDryRunSharedDedicated(t *testing.T) {
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
		mappings = append(mappings, tenantupgrade.Mapping{LegacySchool: school.School, SourcePool: poolName, SourceMarker: marker,
			Tenant: id(0), Organization: id(1), School: id(2), Campus: id(3), Timezone: "UTC", Locale: "en"})
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
	selection := tenantupgrade.SourceSelection{Version: 2, Schools: references.Schools, Documents: tenantupgrade.SourceDocumentEvidence{
		Present: true, Version: 2, Schools: reconciled.Schools, References: int64(reconciled.References), Documents: int64(reconciled.Documents),
		ContentBytes: reconciled.ContentBytes, ReferencesSHA256: reconciled.ReferencesSHA256, InventorySHA256: reconciled.InventorySHA256, SHA256: reconciled.SHA256,
	}}
	keys := make([]string, 0, 41)
	for _, definition := range modules.Definitions() {
		if definition.Migration != nil {
			keys = append(keys, definition.Key)
		}
	}
	source := tenantupgrade.InventorySource(t.Context(), settings, keys, selection)
	if !source.ReadyForCopy {
		t.Fatalf("fixture source evidence not ready: %#v", source)
	}
	manifest.SourceSHA256 = source.SHA256
	upgrade := tenantupgrade.NewUpgrade(settings, keys, manifest, selection)
	report, err := upgrade.DryRun(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !report.Complete || !report.ReadyForPrepare || report.Schools != len(fixture.Schools) || report.Pools != 2 || report.SourceSHA256 != source.SHA256 {
		t.Fatalf("mixed shared/dedicated source did not pass dry-run: %#v", report)
	}
	rehearsal, err := upgrade.RehearseSQLFence(t.Context())
	if err != nil || !rehearsal.Complete || rehearsal.Classification != "legacy-sql-fence-rehearsal" ||
		rehearsal.Pools != 2 || rehearsal.SourceSHA256 != source.SHA256 || rehearsal.ManifestSHA256 != report.ManifestSHA256 {
		t.Fatalf("fenced rehearsal did not reproduce the approved mixed source: %#v %v", rehearsal, err)
	}
	t.Run("native rehearsal command", func(t *testing.T) {
		dir := t.TempDir()
		writeInput := func(name string, value any) string {
			t.Helper()
			path := filepath.Join(dir, name)
			data, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			return path
		}
		binary := filepath.Join(dir, "tenant-upgrade")
		build := exec.CommandContext(t.Context(), "go", "build", "-buildvcs=false", "-o", binary, "../../cmd/tenant-upgrade")
		if err := build.Run(); err != nil {
			t.Fatal("could not build native rehearsal command", err)
		}
		command := exec.CommandContext(t.Context(), binary, "-mode=rehearse-sql-fence",
			"-manifest="+writeInput("mapping.json", manifest), "-selection="+writeInput("selection.json", selection))
		command.Dir = dir
		command.Env = append(os.Environ(), "APP_KEY=12345678901234567890123456789012")
		output, err := command.Output()
		if err != nil {
			t.Fatal("compiled rehearsal command failed", err)
		}
		expected, err := json.Marshal(rehearsal)
		if err != nil || strings.TrimSpace(string(output)) != string(expected) {
			t.Fatal("native rehearsal did not return the expected redacted JSON")
		}
	})
	manifest.SourceSHA256 = strings.Repeat("b", 64)
	rehearsal, err = tenantupgrade.NewUpgrade(settings, keys, manifest, selection).RehearseSQLFence(t.Context())
	if err != nil || rehearsal.Complete || rehearsal.SourceSHA256 != "" || rehearsal.ManifestSHA256 != "" ||
		!slices.Contains(rehearsal.Issues, tenantupgrade.DryRunIssue{Code: "source_fingerprint_mismatch"}) {
		t.Fatalf("rehearsal accepted changed source evidence: %#v %v", rehearsal, err)
	}
}
