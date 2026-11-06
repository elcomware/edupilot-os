package tenantupgrade

import (
	"context"
	"errors"
	"slices"

	"github.com/elcomware/edupilot/platform/database"
)

// DryRunReport is safe to archive as pre-prepare evidence. It deliberately
// omits every legacy school name, target identity and physical marker.
type DryRunReport struct {
	Version         int           `json:"version"`
	Classification  string        `json:"classification"`
	Complete        bool          `json:"complete"`
	ReadyForPrepare bool          `json:"readyForPrepare"`
	Schools         int           `json:"schools"`
	Pools           int           `json:"pools"`
	ManifestSHA256  string        `json:"manifestSha256,omitempty"`
	SourceSHA256    string        `json:"sourceSha256,omitempty"`
	Issues          []DryRunIssue `json:"issues"`
}

type DryRunIssue struct {
	Code string `json:"code"`
}

type sourceInventoryFunc func(context.Context, database.Settings, []string, SourceSelection) SourceInventory
type placementInspectionFunc func(context.Context, database.Settings, []string) PlacementInspection
type sourceMarkerFunc func(context.Context, database.Settings) (map[string]string, error)

// Upgrade owns the immutable inputs used by every later upgrade phase. This
// preflight and rehearsal code never commits a source or target mutation.
type Upgrade struct {
	settings      database.Settings
	workspaceKeys []string
	manifest      Manifest
	selection     SourceSelection
	inventory     sourceInventoryFunc
	placement     placementInspectionFunc
	markers       sourceMarkerFunc
}

func NewUpgrade(settings database.Settings, workspaceKeys []string, manifest Manifest, selection SourceSelection) Upgrade {
	settings.Dedicated = slices.Clone(settings.Dedicated)
	manifest.Schools = slices.Clone(manifest.Schools)
	selection.Schools = slices.Clone(selection.Schools)
	return Upgrade{
		settings: settings, workspaceKeys: slices.Clone(workspaceKeys), manifest: manifest, selection: selection,
	}.withSnapshots(readSnapshots(openInspectionDB))
}

func (u Upgrade) withSnapshots(snapshot sourceSnapshots) Upgrade {
	u.inventory = func(ctx context.Context, settings database.Settings, keys []string, selection SourceSelection) SourceInventory {
		return inventorySource(ctx, settings, keys, selection, snapshot)
	}
	u.placement = func(ctx context.Context, settings database.Settings, keys []string) PlacementInspection {
		return inspectPlacement(ctx, settings, keys, snapshot)
	}
	u.markers = func(ctx context.Context, settings database.Settings) (map[string]string, error) {
		return inspectSourceMarkers(ctx, settings, snapshot)
	}
	return u
}

func (u Upgrade) DryRun(ctx context.Context) (DryRunReport, error) {
	if err := ctx.Err(); err != nil {
		return DryRunReport{}, err
	}
	report := DryRunReport{Version: 1, Classification: "legacy-upgrade-dry-run", Schools: len(u.manifest.Schools), Issues: []DryRunIssue{}}
	problem := func(code string) { report.Issues = append(report.Issues, DryRunIssue{Code: code}) }
	manifestSHA, err := u.manifest.Fingerprint()
	if err != nil {
		problem("invalid_manifest")
		return report, nil
	}
	if err := u.selection.Validate(); err != nil || u.selection.Version != 2 || !u.selection.Documents.Present || u.selection.Documents.Version != 2 {
		problem("source_not_reconciled")
		return report, nil
	}
	legacySchools := make([]string, len(u.manifest.Schools))
	for i, mapping := range u.manifest.Schools {
		legacySchools[i] = mapping.LegacySchool
	}
	slices.Sort(legacySchools)
	if !slices.Equal(legacySchools, u.selection.Schools) {
		problem("source_school_mismatch")
		return report, nil
	}
	source := u.inventory(ctx, u.settings, u.workspaceKeys, u.selection)
	if err := ctx.Err(); err != nil {
		return DryRunReport{}, err
	}
	report.Pools = len(source.Pools)
	if source.Version != 2 || source.Classification != "frozen-source-sql-objects" || !source.Complete || !source.ReadyForCopy ||
		source.Schools != len(legacySchools) || len(source.Issues) != 0 || source.SHA256 == "" {
		problem("source_inventory_incomplete")
		return report, nil
	}
	if source.SHA256 != u.manifest.SourceSHA256 {
		problem("source_fingerprint_mismatch")
		return report, nil
	}
	placement := u.placement(ctx, u.settings, u.workspaceKeys)
	if err := ctx.Err(); err != nil {
		return DryRunReport{}, err
	}
	if !placement.Complete || len(placement.Issues) != 0 {
		problem("placement_incomplete")
		return report, nil
	}
	expectedPools := make(map[string]string, len(u.manifest.Schools))
	for _, mapping := range u.manifest.Schools {
		expectedPools[mapping.LegacySchool] = mapping.SourcePool
	}
	if len(placement.Schools) != len(expectedPools) {
		problem("source_school_mismatch")
		return report, nil
	}
	for _, school := range placement.Schools {
		if expected, ok := expectedPools[school.School]; !ok {
			problem("source_school_mismatch")
			return report, nil
		} else if !school.Identity || !school.Pinned || school.ExpectedPool != expected {
			problem("source_pool_mismatch")
			return report, nil
		}
	}
	markers, err := u.markers(ctx, u.settings)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return DryRunReport{}, err
		}
		problem("source_marker_unavailable")
		return report, nil
	}
	for _, mapping := range u.manifest.Schools {
		if actual, ok := markers[mapping.SourcePool]; !ok || actual != mapping.SourceMarker {
			problem("source_marker_mismatch")
			return report, nil
		}
	}
	report.Complete = true
	report.ReadyForPrepare = true
	report.ManifestSHA256 = manifestSHA
	report.SourceSHA256 = source.SHA256
	return report, nil
}

func inspectSourceMarkers(ctx context.Context, settings database.Settings, snapshot sourceSnapshots) (map[string]string, error) {
	if err := settings.Validate(); err != nil {
		return nil, errors.New("source marker configuration unavailable")
	}
	result := make(map[string]string, len(settings.Nodes()))
	seen := map[string]bool{}
	for _, node := range settings.Nodes() {
		tx, release, err := snapshot(ctx, node)
		if err != nil {
			return nil, errors.New("source marker unavailable")
		}
		var singleton int
		var marker string
		err = tx.QueryRowContext(ctx, "SELECT singleton,pool_id FROM edu_pool_marker WHERE singleton=1").Scan(&singleton, &marker)
		release()
		if err != nil || singleton != 1 || !legacyMarkerValid(marker) || seen[marker] {
			return nil, errors.New("source marker unavailable")
		}
		seen[marker] = true
		result[node.Name] = marker
	}
	return result, nil
}
