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
