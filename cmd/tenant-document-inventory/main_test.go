package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/modules/documents/app/services"
	"github.com/elcomware/edupilot/modules/documents/app/services/filesystem"
	"github.com/elcomware/edupilot/platform/access"
	"github.com/elcomware/edupilot/platform/tenantupgrade"
)

func privateManifest(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "private-manifest.json")
	raw := `{"version":1,"sourceSha256":"` + strings.Repeat("a", 64) + `","schools":[{"legacySchool":"north","sourcePool":"sqlite","sourceMarker":"private-marker","tenant":"b0421bcb-122a-4e2d-8294-261864c2e249","organization":"c88e6562-3dd9-4d83-a115-eacd8c791069","school":"558a7688-4335-4b37-b9ca-91b7a5842cc4","campus":"42a8b3dd-1da0-4dcf-a0af-11140e68fa68","timezone":"UTC","locale":"en"}]}`
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
func TestCommandInventoriesActualUploadOfflineWithNoSourceMutation(t *testing.T) {
	root := privateRoot(t)
	manifest := privateManifest(t)
	db := filepath.Join(t.TempDir(), "must-not-exist.sqlite")
	t.Setenv("DB_DATABASE", db)
	store, err := filesystem.Configured(func(key, fallback string) string {
		if key == "EDUPILOT_DOCUMENTS_ROOT" {
			return root
		}
		return fallback
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := access.WithPrincipal(t.Context(), access.Principal{ID: "fixture-user", School: "north", Permissions: []string{access.DocumentsWrite}})
	id, err := (services.Files{Resolve: func() (services.BlobStore, error) { return store, nil }}).Upload(ctx, "text/plain", []byte("private payload"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "schools", "north", "documents", id)
	before, _ := os.ReadFile(path)
	var output, diagnostics bytes.Buffer
	if code := run([]string{"-manifest", manifest, "-root", root}, &output, &diagnostics); code != 0 {
		t.Fatal(code, diagnostics.String())
	}
	var result filesystem.LocalInventory
	if err = json.Unmarshal(output.Bytes(), &result); err != nil || result.Documents != 1 || result.ContentBytes != 15 || len(result.SHA256) != 64 {
		t.Fatal("invalid aggregate", err)
	}
	for _, secret := range []string{id, root, "private payload", "private-marker"} {
		if strings.Contains(output.String()+diagnostics.String(), secret) {
			t.Fatal("private record metadata/content disclosed")
		}
	}
	output.Reset()
	if code := run([]string{"-manifest", manifest, "-root", root, "-expected-sha256", result.SHA256}, &output, &diagnostics); code != 0 {
		t.Fatal(code, diagnostics.String())
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("changed source")
	}
	if _, err = os.Stat(db); !os.IsNotExist(err) {
		t.Fatal("opened database")
	}
}

func TestCommandReconcilesPreservedReferencesBeforeCopyEvidence(t *testing.T) {
	root := privateRoot(t)
	manifestPath := privateManifest(t)
	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := tenantupgrade.ReadManifest(manifestFile)
	manifestFile.Close()
	if err != nil {
		t.Fatal(err)
	}
	store, err := filesystem.Configured(func(key, fallback string) string {
		if key == "EDUPILOT_DOCUMENTS_ROOT" {
			return root
		}
		return fallback
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := access.WithPrincipal(t.Context(), access.Principal{ID: "fixture-user", School: "north", Permissions: []string{access.DocumentsWrite}})
	id, err := (services.Files{Resolve: func() (services.BlobStore, error) { return store, nil }}).Upload(ctx, "text/plain", []byte("private reconciled payload"))
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := filesystem.InventoryLocal(t.Context(), root, manifest, "")
	if err != nil {
		t.Fatal(err)
	}
	referencesPath := filepath.Join(t.TempDir(), "private-references.json")
	references := `{"version":1,"schools":["north"],"objects":[{"school":"north","id":"` + id + `"}]}`
	if err := os.WriteFile(referencesPath, []byte(references), 0600); err != nil {
		t.Fatal(err)
	}
	var output, diagnostics bytes.Buffer
	args := []string{"-manifest", manifestPath, "-references", referencesPath, "-root", root, "-expected-sha256", baseline.SHA256}
	if code := run(args, &output, &diagnostics); code != 0 {
		t.Fatal(code, diagnostics.String())
	}
	var result filesystem.ReconciledLocalInventory
	if err := json.Unmarshal(output.Bytes(), &result); err != nil || result.Version != 2 || result.References != 1 || result.Documents != 1 || len(result.SHA256) != 64 {
		t.Fatal("invalid reconciled command result", err)
	}
	if strings.Contains(output.String()+diagnostics.String(), id) || strings.Contains(output.String()+diagnostics.String(), referencesPath) {
		t.Fatal("reconciled output disclosed a private object reference")
	}
	exerciseRestoredBucket(t, root, id, manifestPath, referencesPath, baseline.SHA256, output.Bytes())
	if err := os.Remove(filepath.Join(root, "schools", "north", "documents", id)); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	diagnostics.Reset()
	if code := run(args, &output, &diagnostics); code == 0 || output.Len() != 0 {
		t.Fatal("missing referenced object produced copy evidence")
	}
	output.Reset()
	diagnostics.Reset()
	if code := run([]string{"-manifest", manifestPath, "-references", referencesPath, "-root", root}, &output, &diagnostics); code == 0 || output.Len() != 0 {
		t.Fatal("reconciliation silently omitted the independent byte baseline")
	}
}
func TestCommandErrorsArePrivateAndNeverEmitPartialInventory(t *testing.T) {
	root := privateRoot(t)
	manifest := privateManifest(t)
	cases := [][]string{
		{}, {"-manifest", manifest}, {"-manifest", "private-missing-path", "-root", root},
		{"-manifest", manifest, "-root", filepath.Join(root, "missing-private-root")},
		{"-manifest", manifest, "-root", root, "-expected-sha256", strings.Repeat("b", 64)},
		{"-manifest", manifest, "-root", root, "-expected-sha256", "private-expected-digest"},
		{"-private-unknown-flag=private-value"}, {"-manifest", manifest, "-root", root, "private-extra-arg"},
	}
	for _, args := range cases {
		var output, diagnostics bytes.Buffer
		if code := run(args, &output, &diagnostics); code == 0 || output.Len() != 0 {
			t.Fatal("invalid command succeeded or emitted partial output")
		}
		for _, secret := range []string{root, manifest, "private-missing-path", "private-expected-digest", "private-unknown-flag", "private-value", "private-extra-arg", strings.Repeat("b", 64)} {
			if strings.Contains(diagnostics.String(), secret) {
				t.Fatal("command diagnostics disclosed input")
			}
		}
	}
	if err := os.WriteFile(manifest, []byte(`{"version":1,"private-secret":"private-content"}`), 0600); err != nil {
		t.Fatal(err)
	}
	var output, diagnostics bytes.Buffer
	if code := run([]string{"-manifest", manifest, "-root", root}, &output, &diagnostics); code == 0 || output.Len() != 0 || strings.Contains(diagnostics.String(), "private-") {
		t.Fatal("malformed manifest accepted/disclosed")
	}
}

func privateRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	return root
}

type shortInventoryWriter struct{}

func (shortInventoryWriter) Write(value []byte) (int, error) { return len(value) / 2, nil }
func TestCommandDoesNotClaimSuccessForPartialOutput(t *testing.T) {
	root := privateRoot(t)
	var diagnostics bytes.Buffer
	if code := run([]string{"-manifest", privateManifest(t), "-root", root}, shortInventoryWriter{}, &diagnostics); code == 0 {
		t.Fatal("partial output was reported as a complete inventory")
	}
}

func TestCommandRejectsExplicitEmptyExpectedDigest(t *testing.T) {
	for _, option := range [][]string{{"-expected-sha256", ""}, {"-expected-sha256="}} {
		var output, diagnostics bytes.Buffer
		args := append([]string{"-manifest", privateManifest(t), "-root", privateRoot(t)}, option...)
		if code := run(args, &output, &diagnostics); code == 0 || output.Len() != 0 {
			t.Fatal("explicit empty expected digest silently disabled comparison")
		}
	}
}
