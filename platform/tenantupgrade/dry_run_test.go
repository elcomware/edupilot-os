package tenantupgrade

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/platform/database"
)

func TestLegacyUpgradeDryRunBindsReconciledSourceMappingAndPlacement(t *testing.T) {
	selection, err := ReadSourceSelection(strings.NewReader(strings.Replace(validReconciledSourceSelection(), `"east"`, `"north"`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	m := manifest()
	m.SourceSHA256 = strings.Repeat("d", 64)
	u := NewUpgrade(database.Settings{}, []string{"registered"}, m, selection)
	u.inventory = func(context.Context, database.Settings, []string, SourceSelection) SourceInventory {
		return SourceInventory{Version: 2, Classification: "frozen-source-sql-objects", Complete: true, ReadyForCopy: true,
			Schools: 2, Documents: &selection.Documents, Pools: []SourcePoolInventory{{Name: "shared", SHA256: strings.Repeat("c", 64)}},
			Issues: []SourceInventoryIssue{}, SHA256: strings.Repeat("d", 64)}
	}
	u.placement = func(context.Context, database.Settings, []string) PlacementInspection {
		return PlacementInspection{Version: 1, Classification: "inspect-only", Complete: true,
			Pools:   []InspectedPool{{Name: "shared"}},
			Schools: []InspectedSchool{{School: "north", Identity: true, Pinned: true, ExpectedPool: "shared"}, {School: "west", Identity: true, Pinned: true, ExpectedPool: "shared"}},
			Issues:  []InspectionIssue{}}
	}
	u.markers = func(context.Context, database.Settings) (map[string]string, error) {
		return map[string]string{"shared": "physical-A"}, nil
	}
	report, err := u.DryRun(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !report.Complete || !report.ReadyForPrepare || report.Version != 1 || report.Classification != "legacy-upgrade-dry-run" ||
		report.Schools != 2 || report.Pools != 1 || len(report.ManifestSHA256) != 64 || report.SourceSHA256 != m.SourceSHA256 || len(report.Issues) != 0 {
		t.Fatalf("dry-run did not produce bounded prepare evidence: %#v", report)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"north", "west", "physical-A", m.Schools[0].Tenant, m.Schools[0].School} {
		if strings.Contains(string(encoded), private) {
			t.Fatalf("dry-run report disclosed private mapping value %q", private)
		}
	}
}

func TestLegacyUpgradeDryRunRejectsIncompleteOrChangedSource(t *testing.T) {
	selection, err := ReadSourceSelection(strings.NewReader(strings.Replace(validReconciledSourceSelection(), `"east"`, `"north"`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	base := manifest()
	base.SourceSHA256 = strings.Repeat("d", 64)
	tests := map[string]struct {
		change func(*Upgrade)
		code   string
	}{
		"legacy document evidence": {func(u *Upgrade) {
			u.selection = SourceSelection{Version: 1, Schools: []string{"north", "west"}}
		}, "source_not_reconciled"},
		"school set mismatch": {func(u *Upgrade) {
			u.selection.Schools = []string{"north", "south"}
		}, "source_school_mismatch"},
		"changed source digest": {func(u *Upgrade) {
			u.manifest.SourceSHA256 = strings.Repeat("e", 64)
		}, "source_fingerprint_mismatch"},
		"source school count mismatch": {func(u *Upgrade) {
			u.inventory = func(context.Context, database.Settings, []string, SourceSelection) SourceInventory {
				return SourceInventory{Version: 2, Classification: "frozen-source-sql-objects", Complete: true, ReadyForCopy: true,
					Schools: 1, Documents: &selection.Documents, Pools: []SourcePoolInventory{{Name: "shared"}}, Issues: []SourceInventoryIssue{}, SHA256: strings.Repeat("d", 64)}
			}
		}, "source_inventory_incomplete"},
		"changed physical marker": {func(u *Upgrade) {
			u.markers = func(context.Context, database.Settings) (map[string]string, error) {
				return map[string]string{"shared": "other-marker"}, nil
			}
		}, "source_marker_mismatch"},
		"changed placement": {func(u *Upgrade) {
			u.placement = func(context.Context, database.Settings, []string) PlacementInspection {
				return PlacementInspection{Version: 1, Classification: "inspect-only", Complete: true,
					Pools: []InspectedPool{{Name: "shared"}}, Schools: []InspectedSchool{{School: "north", Identity: true, Pinned: true, ExpectedPool: "dedicated"}, {School: "west", Identity: true, Pinned: true, ExpectedPool: "shared"}}, Issues: []InspectionIssue{}}
			}
		}, "source_pool_mismatch"},
		"unpinned placement with existing records": {func(u *Upgrade) {
			u.placement = func(context.Context, database.Settings, []string) PlacementInspection {
				return PlacementInspection{Version: 1, Classification: "inspect-only", Complete: true,
					Pools: []InspectedPool{{Name: "shared"}}, Schools: []InspectedSchool{{School: "north", Identity: true, ExpectedPool: "shared", Rows: []InspectedScopeRows{{Table: "students_records", Rows: 1}}}, {School: "west", Identity: true, Pinned: true, ExpectedPool: "shared"}}, Issues: []InspectionIssue{}}
			}
		}, "source_pool_mismatch"},
		"marker read failure": {func(u *Upgrade) {
			u.markers = func(context.Context, database.Settings) (map[string]string, error) {
				return nil, errors.New("private path")
			}
		}, "source_marker_unavailable"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			u := NewUpgrade(database.Settings{}, []string{"registered"}, base, selection)
			u.inventory = func(context.Context, database.Settings, []string, SourceSelection) SourceInventory {
				return SourceInventory{Version: 2, Classification: "frozen-source-sql-objects", Complete: true, ReadyForCopy: true,
					Schools: 2, Documents: &selection.Documents, Pools: []SourcePoolInventory{{Name: "shared"}}, Issues: []SourceInventoryIssue{}, SHA256: strings.Repeat("d", 64)}
			}
			u.placement = func(context.Context, database.Settings, []string) PlacementInspection {
				return PlacementInspection{Version: 1, Classification: "inspect-only", Complete: true,
					Pools: []InspectedPool{{Name: "shared"}}, Schools: []InspectedSchool{{School: "north", Identity: true, Pinned: true, ExpectedPool: "shared"}, {School: "west", Identity: true, Pinned: true, ExpectedPool: "shared"}}, Issues: []InspectionIssue{}}
			}
			u.markers = func(context.Context, database.Settings) (map[string]string, error) {
				return map[string]string{"shared": "physical-A"}, nil
			}
			tc.change(&u)
			report, err := u.DryRun(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if report.Complete || report.ReadyForPrepare || report.ManifestSHA256 != "" || report.SourceSHA256 != "" || !slices.Contains(report.Issues, DryRunIssue{Code: tc.code}) {
				t.Fatalf("unsafe dry-run accepted or disclosed approval evidence: %#v", report)
			}
		})
	}
}

func TestLegacyUpgradeDryRunHonorsCancellation(t *testing.T) {
	u := NewUpgrade(database.Settings{}, nil, manifest(), SourceSelection{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if report, err := u.DryRun(ctx); !errors.Is(err, context.Canceled) || !reflect.ValueOf(report).IsZero() {
		t.Fatalf("cancelled dry-run continued: %#v %v", report, err)
	}
}
