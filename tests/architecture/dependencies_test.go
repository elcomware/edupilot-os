// Package architecture checks the dependency graph by parsing import declarations.
package architecture

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
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

// Core may depend on shared mechanisms. Business modules cannot import each other.
func TestModuleBoundaries(t *testing.T) {
	base := root(t)
	if _, err := os.Stat(filepath.Join(base, "platform")); !os.IsNotExist(err) {
		t.Fatal("root platform must be absent")
	}
	dirs, err := os.ReadDir(filepath.Join(base, "modules"))
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, dir := range dirs {
		names = append(names, dir.Name())
	}
	if strings.Join(names, ",") != "auth,core,tenancy" {
		t.Fatalf("unexpected module tree: %v", names)
	}
	err = filepath.WalkDir(filepath.Join(base, "modules"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, _ := filepath.Rel(filepath.Join(base, "modules"), path)
		owner := strings.Split(filepath.ToSlash(rel), "/")[0]
		src, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range src.Imports {
			name, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			if strings.HasPrefix(name, project+"bootstrap") || strings.HasPrefix(name, project+"app/providers") || strings.HasPrefix(name, project+"app/schoolapi") {
				t.Errorf("%s imports composition %s", rel, name)
			}
			if target, ok := strings.CutPrefix(name, project+"modules/"); ok {
				other := strings.Split(target, "/")[0]
				if other != owner && other != "core" {
					t.Errorf("%s imports feature %s outside root composition", rel, other)
				}
			}
			if strings.HasPrefix(filepath.ToSlash(rel), "core/scope/") && !strings.HasSuffix(rel, "_test.go") && (strings.HasPrefix(name, project) || strings.HasPrefix(name, "github.com/goravel/") || name == "database/sql" || name == "net/http") {
				t.Errorf("scope must stay independent: %s", name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
