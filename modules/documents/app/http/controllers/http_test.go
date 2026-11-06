package controllers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/modules/documents/app/http/controllers"
	"github.com/elcomware/edupilot/modules/documents/app/services"
	"github.com/elcomware/edupilot/modules/documents/app/services/filesystem"
	"github.com/elcomware/edupilot/platform/access"
)

const collection = "/api/v1/documents/files"

func send(t *testing.T, h http.Handler, method, url, media string, data []byte, p access.Principal, scope string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, url, bytes.NewReader(data))
	if p.ID != "" {
		r = r.WithContext(access.WithPrincipal(r.Context(), p))
	}
	if scope != "" {
		r.Header.Set("X-School-ID", scope)
	}
	if media != "" {
		r.Header.Set("Content-Type", media)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func localHandler(t *testing.T, root string) http.Handler {
	t.Helper()
	store, err := filesystem.Configured(func(key, fallback string) string {
		if key == "EDUPILOT_DOCUMENTS_ROOT" {
			return root
		}
		return fallback
	})
	if err != nil {
		t.Fatal(err)
	}
	return controllers.New(services.Files{Resolve: func() (services.BlobStore, error) { return store, nil }})
}

func TestFileRoundTripAndSchoolIsolation(t *testing.T) {
	root := t.TempDir()
	h := localHandler(t, root)
	north := access.Principal{ID: "teacher-a", School: "north", Permissions: []string{access.DocumentsRead, access.DocumentsWrite}}
	south := access.Principal{ID: "teacher-b", School: "south", Permissions: []string{access.DocumentsRead}}
	content := []byte("Private school notice\n")
	w := send(t, h, "POST", collection, "text/plain; charset=utf-8", content, north, "")
	if w.Code != 201 {
		t.Fatalf("upload %d", w.Code)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || created.ID == "" {
		t.Fatalf("upload response: %v", err)
	}
	item := collection + "/" + created.ID
	w = send(t, h, "GET", item, "", nil, north, "")
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), content) || w.Header().Get("Content-Type") != "text/plain" || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment;") {
		t.Fatalf("download status or metadata: %d", w.Code)
	}
	for _, tc := range []struct {
		p      access.Principal
		scope  string
		status int
	}{{south, "", 404}, {south, "north", 403}, {access.Principal{}, "", 401}} {
		w := send(t, h, "GET", item, "", nil, tc.p, tc.scope)
		if w.Code != tc.status || bytes.Contains(w.Body.Bytes(), content) {
			t.Fatalf("download with scope %q: %d", tc.scope, w.Code)
		}
	}
	diskFile := filepath.Join(root, "schools", "north", "documents", created.ID)
	record, err := os.ReadFile(diskFile)
	if err != nil {
		t.Fatal(err)
	}
	// A copied record must also fail at the other school, even under the same UUID.
	otherDir := filepath.Join(root, "schools", "south", "documents")
	if err := os.MkdirAll(otherDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherDir, created.ID), record, 0600); err != nil {
		t.Fatal(err)
	}
	w = send(t, h, "GET", item, "", nil, south, "")
	if w.Code != 503 || bytes.Contains(w.Body.Bytes(), content) {
		t.Fatalf("misplaced record disclosed: %d", w.Code)
	}
	record[len(record)-1] ^= 1
	if err := os.WriteFile(diskFile, record, 0600); err != nil {
		t.Fatal(err)
	}
	w = send(t, h, "GET", item, "", nil, north, "")
	if w.Code != 503 {
		t.Fatalf("corrupted document returned: %d", w.Code)
	}
}

func TestUploadChecksBeforePublication(t *testing.T) {
	root := filepath.Join(t.TempDir(), "files")
	h := localHandler(t, root)
	writer := access.Principal{ID: "a", School: "north", Permissions: []string{access.DocumentsWrite}}
	reader := access.Principal{ID: "b", School: "north", Permissions: []string{access.DocumentsRead}}
	for _, tc := range []struct {
		p      access.Principal
		media  string
		body   []byte
		status int
	}{
		{reader, "text/plain", []byte("notice"), 403}, {writer, "text/html", []byte("<html>private</html>"), 415},
		{writer, "image/jpeg", []byte("not a JPEG"), 415}, {writer, "text/plain", nil, 400},
		{writer, "text/plain", bytes.Repeat([]byte("a"), services.MaxFileSize+1), 413},
	} {
		if w := send(t, h, "POST", collection, tc.media, tc.body, tc.p, ""); w.Code != tc.status {
			t.Fatalf("upload %q = %d, want %d", tc.media, w.Code, tc.status)
		}
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("rejected upload touched the document directory")
	}
	for _, target := range []string{collection + "/../../secret", collection + "/invalid", collection + "/%2e%2e%2fsecret"} {
		if w := send(t, h, "GET", target, "", nil, reader, ""); w.Code != 404 {
			t.Fatalf("invalid item %q = %d", target, w.Code)
		}
	}
}

func TestLocalStorageRefusesSchoolSymlink(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "schools"), 0700); err != nil {
		t.Fatal(err)
	}
	external := t.TempDir()
	if err := os.Symlink(external, filepath.Join(root, "schools", "north")); err != nil {
		t.Skip("filesystem does not support symlinks")
	}
	h := localHandler(t, root)
	writer := access.Principal{ID: "a", School: "north", Permissions: []string{access.DocumentsWrite}}
	if w := send(t, h, "POST", collection, "text/plain", []byte("private"), writer, ""); w.Code != 503 {
		t.Fatalf("school symlink accepted: %d", w.Code)
	}
	entries, err := os.ReadDir(external)
	if err != nil || len(entries) != 0 {
		t.Fatalf("escaped local root: %v", err)
	}
}
