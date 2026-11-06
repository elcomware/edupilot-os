package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/elcomware/edupilot-os/modules/documents/adapters/outbound/blob"
	"github.com/elcomware/edupilot-os/platform/tenantupgrade"
)

func verifyLocalRestore(upgrade tenantupgrade.Upgrade, manifest tenantupgrade.Manifest, selection tenantupgrade.SourceSelection, referencesPath, root string, output, diagnostics io.Writer) int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	refs, err := openPrivateRegular(referencesPath)
	if err != nil {
		fmt.Fprintln(diagnostics, "Cannot read private object references.")
		return 1
	}
	defer refs.Close()
	references, err := tenantupgrade.ReadObjectReferences(refs)
	if err != nil || selection.Version != 2 {
		fmt.Fprintln(diagnostics, "Invalid restore evidence.")
		return 1
	}
	actual, err := blob.ReconcileLocal(ctx, root, manifest, references, selection.Documents.InventorySHA256)
	if err != nil || actual.SHA256 != selection.Documents.SHA256 {
		fmt.Fprintln(diagnostics, "Restored documents differ from approved evidence.")
		return 1
	}
	report, err := upgrade.DryRun(ctx)
	if err != nil || !report.ReadyForPrepare || !report.Complete {
		fmt.Fprintln(diagnostics, "Restored databases failed source verification.")
		return 1
	}
	// Report only verification of this offline copy, never prepare authority.
	result := struct {
		Classification string `json:"classification"`
		Pools          int    `json:"pools"`
		Documents      int    `json:"documents"`
		SourceSHA256   string `json:"sourceSha256"`
		ManifestSHA256 string `json:"manifestSha256"`
	}{"verified-local-restore", report.Pools, actual.Documents, report.SourceSHA256, report.ManifestSHA256}
	data, err := json.Marshal(result)
	if err != nil {
		return 1
	}
	data = append(data, '\n')
	if n, err := output.Write(data); err != nil || n != len(data) {
		return 1
	}
	return 0
}
