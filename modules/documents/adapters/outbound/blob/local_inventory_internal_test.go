package blob

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/elcomware/edupilot-os/modules/documents/application"
	"github.com/elcomware/edupilot-os/platform/access"
)

func TestLocalInventoryStopsAtAccumulatedContentBudget(t *testing.T) {
	path := t.TempDir()
	if err := os.Chmod(path, 0700); err != nil {
		t.Fatal(err)
	}
	ctx := access.WithPrincipal(t.Context(), access.Principal{ID: "fixture-writer", School: "north", Permissions: []string{access.DocumentsWrite}})
	files := application.Files{Resolve: func() (application.BlobStore, error) { return local{path: path}, nil }}
	id, err := files.Upload(ctx, "text/plain", []byte("private content budget fixture"))
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(filepath.Join(path, "schools", "north", "documents"))
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	// Exercise the next real record after the accumulator reaches its public cap,
	// without allocating or reading a gigabyte in this focused boundary test.
	scan := localInspection{ctx: t.Context(), result: LocalInventory{ContentBytes: MaxInventoryContentBytes}}
	if err = scan.document(root, "north", id); !errors.Is(err, ErrLocalInventoryLimit) {
		t.Fatal("content budget was exceeded", err)
	}
	if scan.result.Documents != 0 || scan.result.ContentBytes != MaxInventoryContentBytes {
		t.Fatal("failed record changed aggregate counters")
	}
}
