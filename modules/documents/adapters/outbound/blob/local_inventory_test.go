package blob_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/elcomware/edupilot-os/modules/documents/adapters/outbound/blob"
	"github.com/elcomware/edupilot-os/modules/documents/application"
	"github.com/elcomware/edupilot-os/platform/access"
	"github.com/elcomware/edupilot-os/platform/tenantupgrade"
	"github.com/google/uuid"
)

func inventoryManifest() tenantupgrade.Manifest {
	m := tenantupgrade.Manifest{Version: 1, SourceSHA256: strings.Repeat("a", 64)}
	for i, school := range []string{"north", "west"} {
		id := func(n int) string { return fmt.Sprintf("%08x-0000-4000-8000-000000000000", i*4+n+1) }
		m.Schools = append(m.Schools, tenantupgrade.Mapping{LegacySchool: school, SourcePool: "shared", SourceMarker: "physical-a", Tenant: id(0), Organization: id(1), School: id(2), Campus: id(3), Timezone: "UTC", Locale: "en"})
	}
	return m
}
func uploadInventoryFile(t *testing.T, root, school string, content []byte) string {
	t.Helper()
	store, err := blob.Configured(func(key, fallback string) string {
		if key == "EDUPILOT_DOCUMENTS_ROOT" {
			return root
		}
		return fallback
	})
	if err != nil {
		t.Fatal(err)
	}
	files := application.Files{Resolve: func() (application.BlobStore, error) { return store, nil }}
	ctx := access.WithPrincipal(t.Context(), access.Principal{ID: "fixture-writer", School: school, Permissions: []string{access.DocumentsWrite}})
	id, err := files.Upload(ctx, "text/plain", content)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(root, "schools", school, "documents", id)
}
func treeState(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		value := fmt.Sprintf("%v:%v:%v", info.Mode(), info.Size(), info.ModTime().UnixNano())
		if info.Mode().IsRegular() {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value += fmt.Sprintf(":%x", sha256.Sum256(raw))
		}
		result[path] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func TestLocalInventoryUsesUploadedRecordsAndIsDeterministicReadOnly(t *testing.T) {
	root := privateRoot(t)
	uploadInventoryFile(t, root, "west", []byte("private-west-content\n"))
	uploadInventoryFile(t, root, "north", []byte("private-north-content\n"))
	before := treeState(t, root)
	m := inventoryManifest()
	result, err := blob.InventoryLocal(t.Context(), root, m, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != 1 || result.Schools != 2 || result.Documents != 2 || result.ContentBytes != 43 || len(result.SHA256) != 64 {
		t.Fatalf("invalid aggregate: %+v", result)
	}
	m.Schools[0], m.Schools[1] = m.Schools[1], m.Schools[0]
	again, err := blob.InventoryLocal(t.Context(), root, m, result.SHA256)
	if err != nil || again != result {
		t.Fatal("unstable enumeration/mapping order", err)
	}
	if !reflect.DeepEqual(before, treeState(t, root)) {
		t.Fatal("inventory modified source tree")
	}
}
func TestLocalInventoryEmptyRequiresExistingValidPrivateRoot(t *testing.T) {
	m := inventoryManifest()
	root := privateRoot(t)
	empty, err := blob.InventoryLocal(t.Context(), root, m, "")
	if err != nil || empty.Documents != 0 || empty.ContentBytes != 0 || len(empty.SHA256) != 64 {
		t.Fatal("empty root failed", err)
	}
	if empty.SHA256 != "d952cd7ad20baddf48d73a2a71643ad097d3671db01389791c9f817d2ed0a022" {
		t.Fatal("version 1 empty aggregate framing changed")
	}
	if err := os.Mkdir(filepath.Join(root, "schools"), 0700); err != nil {
		t.Fatal(err)
	}
	again, err := blob.InventoryLocal(t.Context(), root, m, empty.SHA256)
	if err != nil || empty != again {
		t.Fatal("empty aggregate depends on incidental directories", err)
	}
	for _, path := range []string{"", filepath.Join(root, "missing")} {
		result, err := blob.InventoryLocal(t.Context(), path, m, "")
		if err == nil || !reflect.ValueOf(result).IsZero() {
			t.Fatal("missing root accepted or partial output")
		}
	}
	if _, err := os.Stat(filepath.Join(root, "missing")); !os.IsNotExist(err) {
		t.Fatal("missing root created")
	}
}
func TestLocalInventoryRejectsOmissionAndKeepsSourceComponentIndependent(t *testing.T) {
	root := privateRoot(t)
	path := uploadInventoryFile(t, root, "north", []byte("private-content\n"))
	m := inventoryManifest()
	complete, err := blob.InventoryLocal(t.Context(), root, m, "")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	omitted, err := blob.InventoryLocal(t.Context(), root, m, "")
	if err != nil || omitted.SHA256 == complete.SHA256 || omitted.Documents != 0 {
		t.Fatal("omission not reflected", err)
	}
	result, err := blob.InventoryLocal(t.Context(), root, m, complete.SHA256)
	if err == nil || !reflect.ValueOf(result).IsZero() {
		t.Fatal("expected digest accepted omission or partial result")
	}
	m.SourceSHA256 = complete.SHA256
	m.Schools[0].Tenant = "00000009-0000-4000-8000-000000000000"
	m.Schools[0].Timezone = "Europe/Paris"
	m.Schools[0].Locale = "fr"
	unchanged, err := blob.InventoryLocal(t.Context(), root, m, omitted.SHA256)
	if err != nil || unchanged != omitted {
		t.Fatal("component digest depends on aggregate source hash or target mapping", err)
	}
	m.Schools[1].LegacySchool = "south"
	changed, err := blob.InventoryLocal(t.Context(), root, m, "")
	if err != nil || changed.SHA256 == omitted.SHA256 || changed.Schools != omitted.Schools {
		t.Fatal("empty approved school membership omitted from digest", err)
	}
	for _, digest := range []string{"BAD", strings.Repeat("a", 65), strings.Repeat("A", 64), strings.Repeat("b", 64)} {
		result, err := blob.InventoryLocal(t.Context(), root, m, digest)
		if err == nil || !reflect.ValueOf(result).IsZero() {
			t.Fatal("bad digest accepted")
		}
	}
}
func TestLocalInventoryRejectsMalformedOrForeignEntriesWithoutDisclosure(t *testing.T) {
	cases := map[string]func(*testing.T, string, string){
		"corrupt": func(t *testing.T, root, path string) {
			raw, _ := os.ReadFile(path)
			raw[len(raw)-1] ^= 1
			mustWrite(t, path, raw, 0600)
		},
		"moved uuid": func(t *testing.T, root, path string) {
			if err := os.Rename(path, filepath.Join(filepath.Dir(path), uuid.NewString())); err != nil {
				t.Fatal(err)
			}
		},
		"moved school": func(t *testing.T, root, path string) {
			dest := filepath.Join(root, "schools", "west", "documents")
			mustMkdir(t, dest)
			raw, _ := os.ReadFile(path)
			mustWrite(t, filepath.Join(dest, filepath.Base(path)), raw, 0600)
		},
		"foreign school": func(t *testing.T, root, path string) {
			uploadInventoryFile(t, root, "foreign-school-secret", []byte("other-private-content"))
		},
		"root extra": func(t *testing.T, root, path string) {
			mustWrite(t, filepath.Join(root, "orphan-secret"), []byte("secret-body"), 0600)
		},
		"school extra": func(t *testing.T, root, path string) {
			mustWrite(t, filepath.Join(root, "schools", "north", "extra-secret"), nil, 0600)
		},
		"missing documents directory": func(t *testing.T, root, path string) { mustMkdir(t, filepath.Join(root, "schools", "west")) },
		"temporary": func(t *testing.T, root, path string) {
			mustWrite(t, filepath.Join(filepath.Dir(path), ".upload-secret"), []byte("temporary-content"), 0600)
		},
		"unknown filename": func(t *testing.T, root, path string) {
			mustWrite(t, filepath.Join(filepath.Dir(path), "private-not-a-uuid"), []byte("secret-body"), 0600)
		},
		"nested directory": func(t *testing.T, root, path string) {
			mustMkdir(t, filepath.Join(filepath.Dir(path), uuid.NewString()))
		},
		"record symlink": func(t *testing.T, root, path string) {
			if err := os.Symlink(path, filepath.Join(filepath.Dir(path), uuid.NewString())); err != nil {
				t.Fatal(err)
			}
		},
		"school symlink": func(t *testing.T, root, path string) {
			if err := os.Symlink(filepath.Join(root, "schools", "north"), filepath.Join(root, "schools", "west")); err != nil {
				t.Fatal(err)
			}
		},
		"oversized": func(t *testing.T, root, path string) {
			if err := os.Truncate(path, int64(application.MaxRecordSize)+1); err != nil {
				t.Fatal(err)
			}
		},
		"bad envelope": func(t *testing.T, root, path string) { mustWrite(t, path, []byte("private-not-an-envelope"), 0600) },
		"valid envelope invalid media": func(t *testing.T, root, path string) {
			content := []byte("<html>private html content</html>")
			key := "schools/north/documents/" + filepath.Base(path)
			sum := sha256.Sum256(append(append([]byte(key), 0), content...))
			raw := append([]byte("EDUPILOT-DOC/1\n"), sum[:]...)
			mustWrite(t, path, append(raw, content...), 0600)
		},
		"public file": func(t *testing.T, root, path string) {
			if err := os.Chmod(path, 0644); err != nil {
				t.Fatal(err)
			}
		},
		"public root": func(t *testing.T, root, path string) {
			if err := os.Chmod(root, 0755); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			root := privateRoot(t)
			path := uploadInventoryFile(t, root, "north", []byte("private-content-secret"))
			mutate(t, root, path)
			before := treeState(t, root)
			result, err := blob.InventoryLocal(t.Context(), root, inventoryManifest(), "")
			if err == nil || !reflect.ValueOf(result).IsZero() {
				t.Fatal("invalid tree accepted or partial result")
			}
			if !reflect.DeepEqual(before, treeState(t, root)) {
				t.Fatal("rejected inspection modified source tree")
			}
			for _, secret := range []string{root, filepath.Base(path), "secret", "foreign-school", hex.EncodeToString(sha256.New().Sum(nil))} {
				if strings.Contains(err.Error(), secret) {
					t.Fatal("private input disclosed in diagnostics")
				}
			}
		})
	}
}
func TestLocalInventoryRejectsRootSymlinkInvalidManifestAndCancelledContext(t *testing.T) {
	root := privateRoot(t)
	link := filepath.Join(t.TempDir(), "private-link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	if _, err := blob.InventoryLocal(t.Context(), link, inventoryManifest(), ""); err == nil {
		t.Fatal("root symlink accepted")
	}
	m := inventoryManifest()
	m.Schools[0].LegacySchool = "../private-path"
	if _, err := blob.InventoryLocal(t.Context(), root, m, ""); err == nil {
		t.Fatal("unapproved mapping accepted")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if result, err := blob.InventoryLocal(ctx, root, inventoryManifest(), ""); err == nil || !reflect.ValueOf(result).IsZero() {
		t.Fatal("cancelled inventory produced aggregate")
	}
}
func mustWrite(t *testing.T, path string, content []byte, mode fs.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatal(err)
	}
}
func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
}

// A digest must represent actual complete bytes, not a concatenation ambiguity.
func TestLocalInventoryContentChangesAffectDigest(t *testing.T) {
	root := privateRoot(t)
	path := uploadInventoryFile(t, root, "north", []byte("first-private-content"))
	m := inventoryManifest()
	before, err := blob.InventoryLocal(t.Context(), root, m, "")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	header := len("EDUPILOT-DOC/1\n") + sha256.Size
	content := bytes.Repeat([]byte("b"), len(raw)-header)
	key := "schools/north/documents/" + filepath.Base(path)
	sum := sha256.Sum256(append(append([]byte(key), 0), content...))
	copy(raw[len("EDUPILOT-DOC/1\n"):header], sum[:])
	copy(raw[header:], content)
	mustWrite(t, path, raw, 0600)
	after, err := blob.InventoryLocal(t.Context(), root, m, "")
	if err != nil || before.SHA256 == after.SHA256 {
		t.Fatal("verified changed bytes not bound", err)
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

func TestLocalInventoryBoundsDirectoryEnumeration(t *testing.T) {
	root := privateRoot(t)
	for n := 0; n <= blob.MaxInventoryEntries; n++ {
		mustWrite(t, filepath.Join(root, fmt.Sprintf("unexpected-%05d", n)), nil, 0600)
	}
	result, err := blob.InventoryLocal(t.Context(), root, inventoryManifest(), "")
	if !errors.Is(err, blob.ErrLocalInventoryLimit) || !reflect.ValueOf(result).IsZero() {
		t.Fatal("directory enumeration did not enforce its cap", err)
	}
}
func TestLocalInventoryBoundsDocumentCount(t *testing.T) {
	root := privateRoot(t)
	dir := filepath.Join(root, "schools", "north", "documents")
	mustMkdir(t, dir)
	content := []byte("bounded inventory notice\n")
	for n := 0; n <= blob.MaxInventoryDocuments; n++ {
		id := fmt.Sprintf("%08x-1111-4111-8111-111111111111", n)
		key := "schools/north/documents/" + id
		sum := sha256.Sum256(append(append([]byte(key), 0), content...))
		raw := append([]byte("EDUPILOT-DOC/1\n"), sum[:]...)
		mustWrite(t, filepath.Join(dir, id), append(raw, content...), 0600)
	}
	result, err := blob.InventoryLocal(t.Context(), root, inventoryManifest(), "")
	if !errors.Is(err, blob.ErrLocalInventoryLimit) || !reflect.ValueOf(result).IsZero() {
		t.Fatal("document count did not enforce its cap", err)
	}
}

func TestLocalInventoryRejectsRawParentComponentsBeforePathCleaning(t *testing.T) {
	base := privateRoot(t)
	clean := filepath.Join(base, "entry", "backup")
	physical := filepath.Join(base, "foreign", "backup")
	mustMkdir(t, clean)
	mustMkdir(t, physical)
	mustMkdir(t, filepath.Join(base, "foreign", "child"))
	uploadInventoryFile(t, physical, "north", []byte("different private physical source"))
	if err := os.Symlink(filepath.Join(base, "foreign", "child"), filepath.Join(base, "entry", "link")); err != nil {
		t.Fatal(err)
	}
	// Do not construct these inputs with filepath.Join: it would erase the
	// ambiguity under test before the inventory API receives the original path.
	for _, path := range []string{base + "/entry/link/../backup", clean + "/missing-component/../"} {
		result, err := blob.InventoryLocal(t.Context(), path, inventoryManifest(), "")
		if err == nil || !reflect.ValueOf(result).IsZero() {
			t.Fatal("lexically cleaned path silently selected a different existing root")
		}
	}
}
