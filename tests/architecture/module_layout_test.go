package architecture

import (
	"io/fs"
	"path/filepath"
	"testing"
)

// Keep the framework vocabulary consistent for both existing and newly generated modules.
func TestModuleDirectoryConvention(t *testing.T) {
	forbidden := map[string]bool{"adapters": true, "ports": true, "inbound": true, "outbound": true, "domain": true, "application": true}
	err := filepath.WalkDir(filepath.Join(root(t), "modules"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && forbidden[entry.Name()] {
			t.Errorf("unexpected module directory %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
