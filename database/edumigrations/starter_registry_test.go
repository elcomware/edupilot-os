package edumigrations_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/elcomware/edupilot-os/bootstrap"
	"github.com/elcomware/edupilot-os/database/edumigrations"
	"github.com/elcomware/edupilot-os/platform/database"
	schemaContract "github.com/goravel/framework/contracts/database/schema"
)

type starterRegistrySnapshot struct {
	Baseline      string
	NativeSources map[string]string
	Registrations []edumigrations.StarterMigrationDescriptor
}

func loadStarterRegistry(t *testing.T) starterRegistrySnapshot {
	t.Helper()
	data, err := os.ReadFile("testdata/starter/registrations.json")
	if err != nil {
		t.Fatalf("registered starter snapshot required: %v", err)
	}
	var snapshot starterRegistrySnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Baseline != "049c502536c43f8cefb4d804fab987e6d1855f39" {
		t.Fatal("starter baseline changed without review")
	}
	for path, want := range snapshot.NativeSources {
		data, err := os.ReadFile(filepath.Join("../..", path))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != want {
			t.Fatalf("native starter migration changed: %s", path)
		}
	}
	return snapshot
}

func validateStarterRegistry(snapshot starterRegistrySnapshot, settings database.Settings, actual []schemaContract.Migration) ([]schemaContract.Migration, error) {
	allowed := map[string]bool{}
	for _, d := range snapshot.Registrations {
		allowed[strings.SplitN(d.Signature, "__", 2)[0]] = true
	}
	active := map[string]bool{settings.Control.Name: true}
	for _, node := range settings.Dedicated {
		active[node.Name] = true
	}
	expected := []edumigrations.StarterMigrationDescriptor{}
	for _, d := range snapshot.Registrations {
		if d.Connection == "" {
			expected = append(expected, d)
			continue
		}
		if d.Connection == "sqlite" {
			d.Connection = settings.Control.Name
			d.Signature = strings.TrimSuffix(d.Signature, "__sqlite") + "__" + settings.Control.Name
			for _, statements := range [][]edumigrations.StarterSQLStatement{d.ApplyUp, d.ApplyDown} {
				for i := range statements {
					for j, arg := range statements[i].Args {
						if value, ok := arg.(string); ok && strings.HasSuffix(value, "__sqlite") {
							statements[i].Args[j] = strings.TrimSuffix(value, "__sqlite") + "__" + settings.Control.Name
						}
					}
				}
			}
		}
		d.Driver = settings.Control.Driver
		if active[d.Connection] {
			expected = append(expected, d)
		}
	}
	selected := []schemaContract.Migration{}
	got := []edumigrations.StarterMigrationDescriptor{}
	for _, m := range actual {
		if allowed[strings.SplitN(m.Signature(), "__", 2)[0]] {
			selected = append(selected, m)
			got = append(got, edumigrations.DescribeStarterMigration(m))
		}
	}
	if !reflect.DeepEqual(got, expected) {
		for i := 0; i < len(got) && i < len(expected); i++ {
			if !reflect.DeepEqual(got[i], expected[i]) {
				return nil, fmt.Errorf("registered starter descriptor %d (%s) changed: got %#v; want %#v", i, got[i].Signature, got[i], expected[i])
			}
		}
		return nil, fmt.Errorf("registered starter order/count changed: got %d want %d", len(got), len(expected))
	}
	return selected, nil
}

func TestStarterFixtureRegisteredBaseline(t *testing.T) {
	// Missing/changed snapshots fail before any fixture executes native DDL.
	loadStarterRegistry(t)
}

func TestStarterFixtureDetectsChangedRegisteredPlans(t *testing.T) {
	snapshot := loadStarterRegistry(t)
	settings := database.Settings{Control: database.Connection{Name: "sqlite", Driver: "sqlite", Database: filepath.Join(t.TempDir(), "control.sqlite")}}
	t.Setenv("DB_CONNECTION", "sqlite")
	t.Setenv("DB_DATABASE", settings.Control.Database)
	t.Setenv("EDUPILOT_DEDICATED_DATABASES", "[]")
	original := bootstrap.Migrations()
	if _, err := validateStarterRegistry(snapshot, settings, original); err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"down", "placement", "order"} {
		t.Run(change, func(t *testing.T) {
			mutated := append([]schemaContract.Migration{}, original...)
			switch change {
			case "down":
				mutated[2] = edumigrations.CloneStarterMigration(mutated[2], func(p edumigrations.Plan) edumigrations.Plan { p.Down = []string{"DROP TABLE wrong"}; return p }, nil)
			case "placement":
				mutated[1] = edumigrations.CloneStarterMigration(mutated[1], func(p edumigrations.Plan) edumigrations.Plan { p.ControlOnly = !p.ControlOnly; return p }, nil)
			case "order":
				mutated[1], mutated[2] = mutated[2], mutated[1]
			}
			if _, err := validateStarterRegistry(snapshot, settings, mutated); err == nil {
				t.Fatal("changed original registration accepted")
			}
		})
	}
}
