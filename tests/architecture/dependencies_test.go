// Package architecture checks the dependency graph by parsing import declarations.
package architecture

import (
	"encoding/csv"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/modules"
)

const project = "github.com/elcomware/edupilot/"

func root(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("module root unavailable")
		}
		dir = parent
	}
}
func TestHexagonalImports(t *testing.T) {
	base := root(t)
	for _, area := range []string{"modules", "platform"} {
		err := filepath.WalkDir(filepath.Join(base, area), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(base, path)
			parts := strings.Split(filepath.ToSlash(rel), "/")
			src, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			if area == "modules" && len(parts) < 3 {
				return nil
			} // explicit root registry
			for _, imp := range src.Imports {
				name, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					return err
				}
				if forbidden(area, parts, name) {
					t.Errorf("%s imports %s", rel, name)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
func forbidden(area string, parts []string, name string) bool {
	modulePath := project + "modules/"
	if area == "platform" && len(parts) > 1 && parts[1] == "scope" {
		// Scope is the bottom of the authorization dependency graph.
		return strings.HasPrefix(name, project) || strings.HasPrefix(name, "github.com/goravel/") || strings.HasPrefix(name, "gorm.io/") || name == "database/sql" || name == "net/http"
	}
	if area == "platform" {
		return strings.HasPrefix(name, modulePath)
	}
	if strings.HasPrefix(name, modulePath) {
		other := strings.Split(strings.TrimPrefix(name, modulePath), "/")
		if other[0] != parts[1] && other[0] != "modulekit" {
			return len(other) < 3 || other[1] != "app" || other[2] != "contracts"
		}
	}
	if len(parts) < 4 || parts[2] != "app" {
		return false
	}
	switch parts[3] {
	case "contracts", "models":
	case "services":
		// Auth queries read the authoritative control pool. Document storage
		// implements the private blob service using local and S3 drivers.
		if parts[1] == "auth" || (len(parts) > 4 && parts[1] == "documents" && parts[4] == "filesystem") {
			return false
		}
	default:
		return false
	}
	return strings.HasPrefix(name, "github.com/goravel/") || strings.HasPrefix(name, "gorm.io/") ||
		name == "net/http" || name == "database/sql" || strings.Contains(name, "/database/") ||
		strings.HasSuffix(name, "/database") || strings.Contains(name, "/http/controllers") ||
		strings.Contains(name, "/services/filesystem") || strings.HasPrefix(name, project+"app/")

}

func TestEveryManifestEntryIsWiredOnce(t *testing.T) {
	f, err := os.Open(filepath.Join(root(t), "modules", "catalog.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	reader := csv.NewReader(f)
	reader.Comma = '\t'
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	definitions := modules.Definitions()
	registry, err := modules.NewRegistry(nil, definitions)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != len(rows) {
		t.Fatalf("got %d definitions for %d manifest rows", len(definitions), len(rows))
	}
	for _, row := range rows {
		key := row[1]
		d, ok := registry.Find(key)
		if !ok || d.Group != row[0] || d.Title.EN != row[2] || d.Title.FR != row[3] {
			t.Errorf("catalog drift for %s", key)
		}
		if key != "students" {
			service, ok := registry.Entries(key)
			if !ok || service.Store == nil || d.Migration == nil || d.Migration().ID == "" {
				t.Errorf("module has no persistence or migration: %s", key)
			}
		}
	}
}
