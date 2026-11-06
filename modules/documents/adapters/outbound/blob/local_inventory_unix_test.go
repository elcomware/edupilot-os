//go:build unix

package blob_test

import (
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"

	"github.com/elcomware/edupilot/modules/documents/adapters/outbound/blob"
	"github.com/google/uuid"
)

func TestLocalInventoryRejectsFIFOWithoutOpeningIt(t *testing.T) {
	root := privateRoot(t)
	path := uploadInventoryFile(t, root, "north", []byte("private FIFO fixture"))
	fifo := filepath.Join(filepath.Dir(path), uuid.NewString())
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	result, err := blob.InventoryLocal(t.Context(), root, inventoryManifest(), "")
	if err == nil || !reflect.ValueOf(result).IsZero() {
		t.Fatal("non-regular entry accepted")
	}
	if info, e := os.Lstat(fifo); e != nil || info.Mode()&os.ModeNamedPipe == 0 {
		t.Fatal("inventory altered FIFO", e)
	}
}
