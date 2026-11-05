package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsRejectAliasedAndMalformedDedicatedTargets(t *testing.T) {
	root := t.TempDir()
	shared := filepath.Join(root, "control.sqlite")
	if err := os.WriteFile(shared, nil, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "dedicated.sqlite")
	if err := os.Symlink(shared, link); err != nil {
		t.Fatal(err)
	}
	for _, malformed := range []string{
		`[{"tenant":"school-b","driver":"sqlite","database":"` + link + `"}]`,
		`[{"tenant":"school-b","driver":"sqlite","database":"/one"},{"tenant":"school-b","driver":"sqlite","database":"/two"}]`,
		`[{"tenant":"../bad","driver":"sqlite","database":"/one"}]`,
		`[{"tenant":"ok","driver":"sqlite","databse":"typo"}]`,
		`[] false`, `null`,
	} {
		_, err := ReadSettings(func(k, d string) string {
			switch k {
			case "EDUPILOT_DEDICATED_DATABASES":
				return malformed
			case "DB_DATABASE":
				return shared
			}
			return d
		})
		if err == nil {
			t.Errorf("accepted an unsafe database configuration: %s", malformed)
		}
	}
}

func TestPlaceholderNumberingSkipsQuotedQuestionMarks(t *testing.T) {
	input := `SELECT '?', 'who''s ?', "?", ? /* ? */ FROM s WHERE id = ? -- ?
AND school = ?`
	want := `SELECT '?', 'who''s ?', "?", $1 /* ? */ FROM s WHERE id = $2 -- ?
AND school = $3`
	if got := (Pool{Driver: "postgres"}).Bind(input); got != want {
		t.Fatalf("Postgres query:\n%s", got)
	}
	if got := (Pool{Driver: "sqlite"}).Bind(input); got != input {
		t.Fatalf("SQLite query:\n%s", got)
	}
}

func TestMixedConnectionsKeepDistinctDriverCallbacks(t *testing.T) {
	s, err := ReadSettings(func(k, d string) string {
		if k == "EDUPILOT_DEDICATED_DATABASES" {
			return `[{"tenant":"east","driver":"postgres","host":"localhost","database":"school_east","username":"svc","password":"a/b@c"}]`
		}
		return d
	})
	if err != nil {
		t.Fatal(err)
	}
	m := s.Connections()
	pg := m["tenant_east"].(map[string]any)
	sqlite := m["sqlite"].(map[string]any)
	if len(m) != 2 || pg["sslmode"] != "require" || pg["port"] != 5432 ||
		pg["dsn"] == sqlite["dsn"] || pg["via"] == nil || sqlite["via"] == nil {
		t.Errorf("expected two independent configurations with safe defaults")
	}
}
