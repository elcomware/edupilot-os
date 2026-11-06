package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatorIsOfflineAndRedactsInputs(t *testing.T) {
	t.Setenv("DB_DATABASE", filepath.Join(t.TempDir(), "must-stay-absent.sqlite"))
	raw := `{"version":1,"sourceSha256":"` + strings.Repeat("a", 64) + `","schools":[{"legacySchool":"test","sourcePool":"sqlite","sourceMarker":"private-marker","tenant":"b0421bcb-122a-4e2d-8294-261864c2e249","organization":"c88e6562-3dd9-4d83-a115-eacd8c791069","school":"558a7688-4335-4b37-b9ca-91b7a5842cc4","campus":"42a8b3dd-1da0-4dcf-a0af-11140e68fa68","timezone":"UTC","locale":"en"}]}`
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diag bytes.Buffer
	if status := run([]string{"-mode=validate-manifest", "-manifest", path}, &out, &diag); status != 0 {
		t.Fatal(status, diag.String())
	}
	var value result
	if err := json.Unmarshal(out.Bytes(), &value); err != nil || len(value.ManifestSHA256) != 64 || value.Schools != 1 || strings.Contains(out.String(), "private-marker") {
		t.Fatal("invalid redacted summary", err)
	}
	if status := run([]string{"-mode=resume", "-manifest", path}, &out, &diag); status != 2 {
		t.Fatal("unexpected mutation mode")
	}
	if _, err := os.Stat(os.Getenv("DB_DATABASE")); !os.IsNotExist(err) {
		t.Fatal("unexpected database creation", err)
	}
}

func TestInspectPlacementIsExplicitAndMissingDatabaseIsNotCreated(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.sqlite")
	t.Setenv("APP_KEY", "12345678901234567890123456789012")
	t.Setenv("DB_CONNECTION", "sqlite")
	t.Setenv("DB_DATABASE", missing)
	t.Setenv("EDUPILOT_DEDICATED_DATABASES", "[]")
	var out, diag bytes.Buffer
	if status := run([]string{"-mode=inspect-placement"}, &out, &diag); status != 1 {
		t.Fatalf("inspection must report unavailable input: status %d diagnostics %s", status, diag.String())
	}
	if !strings.Contains(out.String(), `"classification":"inspect-only"`) || !strings.Contains(out.String(), `"code":"pool_unavailable"`) {
		t.Fatalf("missing read-only inspection report: %s", out.String())
	}
	if strings.Contains(out.String()+diag.String(), missing) {
		t.Fatal("database path leaked")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("missing database was created: %v", err)
	}
	out.Reset()
	diag.Reset()
	if status := run(nil, &out, &diag); status != 2 {
		t.Fatal("default mode performed inspection")
	}
}

func TestInvalidFlagsKeepOperatorInputOutOfDiagnostics(t *testing.T) {
	for _, args := range [][]string{{"-private-connection-string"}, {"-mode=private-connection-string"}, {"-manifest"}, {"-help"}} {
		var output, diagnostics bytes.Buffer
		wantStatus := 2
		if args[0] == "-help" {
			wantStatus = 0
		}
		if got := run(args, &output, &diagnostics); got != wantStatus {
			t.Fatal("unexpected exit status", got)
		}
		if output.Len() != 0 || strings.Contains(diagnostics.String(), "private-connection-string") ||
			!strings.Contains(diagnostics.String(), "validate-manifest") || !strings.Contains(diagnostics.String(), "inspect-placement") || !strings.Contains(diagnostics.String(), "inventory-source") {
			t.Fatal("usage must list supported operations without echoing operator input")
		}
	}
}

func TestSourceInventoryRequiresPrivateSelectionAndNeverCreatesMissingDatabase(t *testing.T) {
	dir := t.TempDir()
	selection := filepath.Join(dir, "source.json")
	if err := os.WriteFile(selection, []byte(`{"version":1,"schools":["east"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "must-not-exist.sqlite")
	t.Setenv("APP_KEY", "12345678901234567890123456789012")
	t.Setenv("DB_CONNECTION", "sqlite")
	t.Setenv("DB_DATABASE", missing)
	t.Setenv("EDUPILOT_DEDICATED_DATABASES", "[]")
	var out, diag bytes.Buffer
	if status := run([]string{"-mode=inventory-source", "-selection", selection}, &out, &diag); status != 1 {
		t.Fatalf("missing frozen source succeeded: %d %s", status, diag.String())
	}
	if !strings.Contains(out.String(), `"classification":"frozen-source-sql"`) || !strings.Contains(out.String(), `"code":"pool_unavailable"`) {
		t.Fatalf("missing safe failure report: %s", out.String())
	}
	if strings.Contains(out.String()+diag.String(), missing) || strings.Contains(out.String()+diag.String(), "east") {
		t.Fatal("source inventory leaked a path or school identifier")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("source inventory created a missing database")
	}
	if err := os.Chmod(selection, 0644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	diag.Reset()
	if status := run([]string{"-mode=inventory-source", "-selection", selection}, &out, &diag); status != 1 || out.Len() != 0 || strings.Contains(diag.String(), selection) {
		t.Fatal("public source selection was accepted or disclosed")
	}
}

func TestUpgradeDryRunIsExplicitReadOnlyAndRedacted(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "private-manifest.json")
	manifest := `{"version":1,"sourceSha256":"` + strings.Repeat("d", 64) + `","schools":[{"legacySchool":"private-school","sourcePool":"sqlite","sourceMarker":"private-marker","tenant":"b0421bcb-122a-4e2d-8294-261864c2e249","organization":"c88e6562-3dd9-4d83-a115-eacd8c791069","school":"558a7688-4335-4b37-b9ca-91b7a5842cc4","campus":"42a8b3dd-1da0-4dcf-a0af-11140e68fa68","timezone":"UTC","locale":"en"}]}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	selectionPath := filepath.Join(dir, "private-selection.json")
	selection := `{"version":2,"schools":["private-school"],"documents":{"version":2,"schools":1,"references":0,"documents":0,"contentBytes":0,"referencesSha256":"` + strings.Repeat("a", 64) + `","inventorySha256":"` + strings.Repeat("b", 64) + `","sha256":"09f8ce145f5872b07ab4b09270b3b64d4715b70904dbb58e7b5c199bb56892d0"}}`
	if err := os.WriteFile(selectionPath, []byte(selection), 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "must-not-exist.sqlite")
	t.Setenv("APP_KEY", "12345678901234567890123456789012")
	t.Setenv("DB_CONNECTION", "sqlite")
	t.Setenv("DB_DATABASE", missing)
	t.Setenv("EDUPILOT_DEDICATED_DATABASES", "[]")
	var output, diagnostics bytes.Buffer
	status := run([]string{"-mode=dry-run", "-manifest", manifestPath, "-selection", selectionPath}, &output, &diagnostics)
	if status != 1 || !strings.Contains(output.String(), `"classification":"legacy-upgrade-dry-run"`) || !strings.Contains(output.String(), `"code":"source_inventory_incomplete"`) {
		t.Fatalf("dry-run did not return safe failure evidence: %d %s %s", status, output.String(), diagnostics.String())
	}
	for _, private := range []string{missing, "private-school", "private-marker", manifestPath, selectionPath} {
		if strings.Contains(output.String()+diagnostics.String(), private) {
			t.Fatalf("dry-run disclosed private input %q", private)
		}
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("dry-run created the missing database: %v", err)
	}
	output.Reset()
	diagnostics.Reset()
	status = run([]string{"-mode=rehearse-sql-fence", "-manifest", manifestPath, "-selection", selectionPath}, &output, &diagnostics)
	if status != 1 || !strings.Contains(output.String(), `"code":"sql_fence_unavailable"`) ||
		strings.Contains(output.String()+diagnostics.String(), missing) || strings.Contains(output.String(), "Sha256") {
		t.Fatal("missing rehearsal source did not fail privately")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("rehearsal created a missing database", err)
	}
	if err := os.Chmod(selectionPath, 0644); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	diagnostics.Reset()
	if status := run([]string{"-mode=dry-run", "-manifest", manifestPath, "-selection", selectionPath}, &output, &diagnostics); status != 1 || output.Len() != 0 {
		t.Fatal("dry-run accepted a public selection or emitted partial output")
	}
	output.Reset()
	diagnostics.Reset()
	if status := run([]string{"-mode=rehearse-sql-fence", "-manifest", manifestPath, "-selection", selectionPath}, &output, &diagnostics); status != 1 || output.Len() != 0 {
		t.Fatal("rehearsal accepted a public selection")
	}
}

func TestRestoreModeRequiresBothRestoreInputs(t *testing.T) {
	for _, args := range [][]string{
		{"-mode=verify-local-restore", "-manifest=m", "-selection=s", "-references=r"},
		{"-mode=verify-local-restore", "-manifest=m", "-selection=s", "-root=secret-root"},
		{"-mode=dry-run", "-manifest=m", "-selection=s", "-root=secret-root"},
		{"-mode=inspect-placement", "-references="},
	} {
		var out, diag bytes.Buffer
		if run(args, &out, &diag) != 2 || out.Len() != 0 || strings.Contains(diag.String(), "secret-root") {
			t.Fatal("invalid restore options were accepted or disclosed")
		}
	}
}
