package edumigrations_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/elcomware/edupilot-os/modules"
	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/elcomware/edupilot-os/platform/tenantupgrade"
)

func inspectionKeys() []string {
	keys := []string{}
	for _, d := range modules.Definitions() {
		if d.Migration != nil {
			keys = append(keys, d.Key)
		}
	}
	return keys
}
func inspectionSettings(t *testing.T) database.Settings {
	t.Helper()
	s, err := database.ConfiguredSettings()
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func inspectFixture(t *testing.T) tenantupgrade.PlacementInspection {
	t.Helper()
	return tenantupgrade.InspectPlacement(t.Context(), inspectionSettings(t), inspectionKeys())
}
func issueCodes(report tenantupgrade.PlacementInspection) map[string]bool {
	codes := map[string]bool{}
	for _, issue := range report.Issues {
		codes[issue.Code] = true
	}
	return codes
}
func TestLegacyPlacementInspectionFrozenFixtures(t *testing.T) {
	for _, mode := range []string{"shared", "dedicated", "mixed"} {
		t.Run(mode, func(t *testing.T) {
			f := starterFixture(t, mode)
			settings := inspectionSettings(t)
			before := map[string][]byte{}
			for _, node := range settings.Nodes() {
				data, err := os.ReadFile(node.Database)
				if err != nil {
					t.Fatal(err)
				}
				before[node.Database] = data
			}
			report := inspectFixture(t)
			if report.Classification != "inspect-only" || !report.Complete || len(report.Issues) != 0 || len(report.Schools) != 2 {
				t.Fatalf("clean source inspection: %#v", report)
			}
			if len(report.Pools) != len(settings.Nodes()) {
				t.Fatal("pool enumeration incomplete")
			}
			for _, school := range report.Schools {
				if !school.Identity || !school.Pinned || len(school.Rows) != 45 {
					t.Fatalf("school routing/counts incomplete: %#v", school)
				}
			}
			encoded, _ := json.Marshal(report)
			again, _ := json.Marshal(inspectFixture(t))
			if !bytes.Equal(encoded, again) {
				t.Fatal("inspection report is not deterministic")
			}
			for path, data := range before {
				after, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(data, after) {
					t.Fatalf("source database changed: %v", err)
				}
			}
			var pins int
			if err := f.Control.DB.QueryRow("SELECT count(*) FROM edu_tenant_placement").Scan(&pins); err != nil || pins != 2 {
				t.Fatal("inspection changed legacy pins")
			}
		})
	}
}
func TestLegacyPlacementInspectionFindsRoutingDamageWithoutPinning(t *testing.T) {
	f := starterFixture(t, "mixed")
	if _, err := f.Control.DB.Exec("DELETE FROM edu_tenant_placement WHERE tenant_id='east'"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Control.DB.Exec("INSERT INTO students_records VALUES ('orphan','collision','PRIVATE-CONTENT-CANARY','secret','active',1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Dedicated["west"].DB.Exec("INSERT INTO students_records VALUES ('east','collision','PRIVATE-CONTENT-CANARY','secret','active',1)"); err != nil {
		t.Fatal(err)
	}
	report := inspectFixture(t)
	codes := issueCodes(report)
	for _, want := range []string{"unpinned_school", "missing_identity", "unknown_scope", "unexpected_pool_rows", "duplicate_school_ownership"} {
		if !codes[want] {
			t.Errorf("missing %s in %#v", want, report.Issues)
		}
	}
	var pins int
	if err := f.Control.DB.QueryRow("SELECT count(*) FROM edu_tenant_placement").Scan(&pins); err != nil || pins != 1 {
		t.Fatal("inspection pinned an unpinned school")
	}
	encoded, _ := json.Marshal(report)
	if strings.Contains(string(encoded), "PRIVATE-CONTENT-CANARY") || strings.Contains(string(encoded), "password") || strings.Contains(string(encoded), "pool_id") {
		t.Fatal("private data leaked")
	}
}
func TestLegacyPlacementInspectionMarkerAndPinCorruption(t *testing.T) {
	for _, damage := range []string{"wrong-pin", "duplicate-marker", "wrong-owner", "missing-table", "active-lease"} {
		t.Run(damage, func(t *testing.T) {
			f := starterFixture(t, "mixed")
			west := f.Dedicated["west"]
			want := ""
			switch damage {
			case "wrong-pin":
				var marker string
				if err := west.DB.QueryRow("SELECT pool_id FROM edu_pool_marker").Scan(&marker); err != nil {
					t.Fatal(err)
				}
				if _, err := f.Control.DB.Exec("UPDATE edu_tenant_placement SET pool_id=? WHERE tenant_id='east'", marker); err != nil {
					t.Fatal(err)
				}
				want = "wrong_pin_pool"
			case "duplicate-marker":
				var marker string
				if err := f.Control.DB.QueryRow("SELECT pool_id FROM edu_pool_marker").Scan(&marker); err != nil {
					t.Fatal(err)
				}
				if _, err := west.DB.Exec("UPDATE edu_pool_marker SET pool_id=?", marker); err != nil {
					t.Fatal(err)
				}
				want = "duplicate_pool_marker"
			case "wrong-owner":
				if _, err := west.DB.Exec("UPDATE edu_pool_marker SET owner='shared'"); err != nil {
					t.Fatal(err)
				}
				want = "wrong_pool_owner"
			case "missing-table":
				if _, err := west.DB.Exec("DROP TABLE ws_portal"); err != nil {
					t.Fatal(err)
				}
				want = "missing_legacy_table"
			case "active-lease":
				injectStarterFaults(t, f)
				want = "active_outbox_lease"
			}
			report := inspectFixture(t)
			if !issueCodes(report)[want] {
				t.Fatalf("%s not detected: %#v", damage, report.Issues)
			}
			if damage == "missing-table" && report.Complete {
				t.Fatal("partial table inspection marked complete")
			}
		})
	}
}
func TestLegacyPlacementInspectionActualCommand(t *testing.T) {
	starterFixture(t, "mixed")
	t.Setenv("APP_KEY", "12345678901234567890123456789012")
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "tenant-upgrade")
	build := exec.CommandContext(ctx, "go", "build", "-buildvcs=false", "-o", binary, "../../cmd/tenant-upgrade")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build actual command: %v %s", err, output)
	}
	environmentDir := t.TempDir()
	missingEnvDB := filepath.Join(environmentDir, "env-only-missing.sqlite")
	if err := os.WriteFile(filepath.Join(environmentDir, ".env"), []byte("APP_KEY=12345678901234567890123456789012\nDB_CONNECTION=sqlite\nDB_DATABASE="+missingEnvDB+"\nEDUPILOT_DEDICATED_DATABASES=[]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, binary, "-mode=inspect-placement")
	command.Dir = environmentDir
	command.Env = os.Environ()
	output, err := command.Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			t.Fatalf("native command failed: %s", e.Stderr)
		}
		t.Fatal(err)
	}
	var report tenantupgrade.PlacementInspection
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("invalid native report: %v", err)
	}
	if !report.Complete || len(report.Issues) != 0 || !reflect.DeepEqual(report.Schools, inspectFixture(t).Schools) {
		t.Fatal("native command did not inspect configured legacy routes")
	}
	// Removing the process override uses the native .env value; the prior
	// successful run proves process DB_DATABASE outranks that same .env value.
	envCommand := exec.CommandContext(ctx, binary, "-mode=inspect-placement")
	envCommand.Dir = environmentDir
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "DB_DATABASE=") {
			envCommand.Env = append(envCommand.Env, value)
		}
	}
	envOutput, envErr := envCommand.Output()
	if envErr == nil {
		t.Fatal("native .env database setting was ignored")
	}
	var envReport tenantupgrade.PlacementInspection
	if err := json.Unmarshal(envOutput, &envReport); err != nil || !issueCodes(envReport)["pool_unavailable"] {
		t.Fatalf("native .env inspection report: %v", err)
	}
	if strings.Contains(string(envOutput), missingEnvDB) {
		t.Fatal("native .env path leaked")
	}
	// Native ro opening never creates an absent target or adjacent journal.
	missing := filepath.Join(t.TempDir(), "missing.sqlite")
	t.Setenv("DB_DATABASE", missing)
	report = inspectFixture(t)
	if report.Complete || !issueCodes(report)["pool_unavailable"] {
		t.Fatal("missing database was accepted")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("inspection created missing source")
	}
}

func TestLegacyPlacementInspectionDistinguishesUnverifiedAndMisplacedControl(t *testing.T) {
	for _, damage := range []string{"misplaced-control", "missing-pool", "ambiguous-marker", "oversized-routing"} {
		t.Run(damage, func(t *testing.T) {
			f := starterFixture(t, "mixed")
			settings := inspectionSettings(t)
			want := ""
			switch damage {
			case "misplaced-control":
				if _, err := f.Dedicated["west"].DB.Exec("CREATE TABLE edu_tenant_placement(tenant_id TEXT,pool_id TEXT)"); err != nil {
					t.Fatal(err)
				}
				want = "unexpected_control_table"
			case "missing-pool":
				if err := os.Remove(settings.Dedicated[0].Database); err != nil {
					t.Fatal(err)
				}
				want = "pin_pool_unverified"
			case "ambiguous-marker":
				var id string
				if err := f.Control.DB.QueryRow("SELECT pool_id FROM edu_pool_marker").Scan(&id); err != nil {
					t.Fatal(err)
				}
				if _, err := f.Dedicated["west"].DB.Exec("UPDATE edu_pool_marker SET pool_id=?", id); err != nil {
					t.Fatal(err)
				}
				want = "ambiguous_pin_pool"
			case "oversized-routing":
				huge := strings.Repeat("PRIVATE-OVERSIZED-", 4096)
				if _, err := f.Control.DB.Exec("UPDATE edu_tenant_placement SET pool_id=? WHERE tenant_id='east'", huge); err != nil {
					t.Fatal(err)
				}
				if _, err := f.Dedicated["west"].DB.Exec("UPDATE edu_pool_marker SET owner=?", huge); err != nil {
					t.Fatal(err)
				}
				want = "invalid_school_pin"
			}
			report := inspectFixture(t)
			codes := issueCodes(report)
			if !codes[want] {
				t.Fatalf("missing %s: %#v", want, report.Issues)
			}
			if damage == "missing-pool" && codes["unknown_pin_pool"] {
				t.Fatal("unreadable source was reported as a definitely unknown pin")
			}
			encoded, _ := json.Marshal(report)
			if strings.Contains(string(encoded), "PRIVATE-OVERSIZED") {
				t.Fatal("oversized routing field leaked")
			}
		})
	}
}

func TestLegacyPlacementInspectionRejectsSQLiteNULTails(t *testing.T) {
	f := starterFixture(t, "mixed")
	if _, err := f.Control.DB.Exec("INSERT INTO students_records VALUES (?, 'nul-record','Private','Student','active',1)", "east\x00wrong"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Control.DB.Exec("INSERT INTO students_records VALUES (CAST(? AS BLOB), 'blob-record','Private','Student','active',1)", "east"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Control.DB.Exec("UPDATE edu_pool_marker SET owner=?,pool_id=pool_id||?", "shared\x00wrong", "\x00wrong"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Control.DB.Exec("UPDATE edu_tenant_placement SET pool_id=pool_id||? WHERE tenant_id='west'", "\x00wrong"); err != nil {
		t.Fatal(err)
	}
	report := inspectFixture(t)
	codes := issueCodes(report)
	for _, code := range []string{"invalid_school_scope", "invalid_pool_marker", "invalid_school_pin"} {
		if !codes[code] {
			t.Errorf("NUL-affixed field was accepted: missing %s", code)
		}
	}
	for _, school := range report.Schools {
		if school.School == "east" {
			for _, rows := range school.Rows {
				if rows.Table == "students_records" && rows.Rows != 2 {
					t.Error("NUL-affixed school was attributed to east")
				}
			}
		}
	}
	encoded, _ := json.Marshal(report)
	if strings.Contains(string(encoded), "wrong") || strings.Contains(string(encoded), `\u0000`) {
		t.Fatal("invalid NUL-affixed fields leaked")
	}
}
