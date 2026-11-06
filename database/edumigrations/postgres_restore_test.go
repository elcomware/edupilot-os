package edumigrations_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/elcomware/edupilot-os/platform/database"
)

// Exercise the real archive tools and the operator binary. The only mutable SQL
// targets in this test are fresh databases allocated by newPostgresDatabaseDrill.
func TestLegacyPostgresArchiveRestore(t *testing.T) {
	drill := newPostgresDatabaseDrill(t)
	for _, tool := range []string{"pg_dump", "pg_restore"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Fatalf("the opted-in PostgreSQL restore drill requires %s on PATH", tool)
		}
	}
	fixture := starterFixtureOn(t, "mixed", "postgres", &drill)
	source, err := database.ConfiguredSettings()
	if err != nil {
		t.Fatal("cannot read the synthetic source settings")
	}
	manifest, selection, refs := approveMixedSource(t, fixture, source)
	digest, err := manifest.Fingerprint()
	if err != nil {
		t.Fatal("could not fingerprint the pre-backup mapping")
	}
	dir := t.TempDir()
	mappingPath := writeDrillInput(t, dir, "mapping.json", manifest)
	selectionPath := writeDrillInput(t, dir, "selection.json", selection)
	refsPath := writeDrillInput(t, dir, "references.json", refs)

	// Use explicit connection inputs. In particular, an inherited PGSERVICE,
	// PGHOSTADDR or PGOPTIONS must never redirect pg_restore elsewhere.
	runTool := func(name string, dbName string, args ...string) {
		t.Helper()
		cmd := exec.CommandContext(drill.ctx, name, append([]string{"--no-password", "--dbname", dbName}, args...)...)
		cmd.Dir = dir
		cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "PGHOST=" + drill.cfg.Host,
			"PGPORT=" + strconv.Itoa(int(drill.cfg.Port)), "PGUSER=" + drill.cfg.User, "PGPASSWORD=" + drill.cfg.Password,
			"PGSSLMODE=" + drill.sslmode, "PGCONNECT_TIMEOUT=10"}
		for key, value := range drill.tlsEnvironment {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
		if err := cmd.Run(); err != nil {
			t.Fatalf("%s failed on a disposable pool: %v", name, err)
		}
	}
	archive := func(node database.Connection) string {
		t.Helper()
		path := filepath.Join(dir, node.Name+".dump")
		runTool("pg_dump", node.Database, "--format=custom", "--file", path)
		return path
	}
	controlArchive, westArchive := archive(source.Control), archive(source.Dedicated[0])
	root := filepath.Join(dir, "restored-documents")
	restoreFixtureDocuments(t, fixture, root)

	control, west := drill.create("restore_control"), drill.create("restore_west")
	if err := west.beforeDDL(drill.ctx, control.db, func() error {
		t.Fatal("a sibling connection reached the restore mutation")
		return nil
	}); err == nil {
		t.Fatal("wrong database passed the ownership check")
	}
	restore := func(target ownedDatabase, path string) {
		t.Helper()
		runTool("pg_restore", target.name, "--single-transaction", "--exit-on-error", "--no-owner", "--no-privileges", "--no-tablespaces", path)
	}
	restore(control, controlArchive)
	target := drill.settings(control, west)
	pools, err := json.Marshal(target.Dedicated)
	if err != nil {
		t.Fatal("cannot encode restored pool settings")
	}
	t.Setenv("DB_DATABASE", control.name)
	t.Setenv("EDUPILOT_DEDICATED_DATABASES", string(pools))
	t.Setenv("APP_KEY", "12345678901234567890123456789012")
	binary := filepath.Join(dir, "tenant-upgrade")
	build := exec.CommandContext(drill.ctx, "go", "build", "-buildvcs=false", "-o", binary, "../../cmd/tenant-upgrade")
	if err := build.Run(); err != nil {
		t.Fatal("cannot build operator command", err)
	}
	verify := func(wantSuccess bool) {
		t.Helper()
		cmd := exec.CommandContext(drill.ctx, binary, "-mode=verify-local-restore", "-manifest="+mappingPath,
			"-selection="+selectionPath, "-references="+refsPath, "-root="+root)
		cmd.Dir = dir
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		if !wantSuccess {
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 1 || stdout.Len() != 0 || stderr.String() != "Restored databases failed source verification.\n" {
				t.Fatal("incomplete or altered SQL restore did not fail with the redacted diagnostic")
			}
			return
		}
		var report struct {
			Classification               string
			Pools, Documents             int
			SourceSHA256, ManifestSHA256 string
		}
		if err != nil || stderr.Len() != 0 || json.Unmarshal(stdout.Bytes(), &report) != nil ||
			report.Classification != "verified-local-restore" || report.Pools != 2 || report.Documents != len(fixture.Blobs) ||
			report.SourceSHA256 != manifest.SourceSHA256 || report.ManifestSHA256 != digest {
			t.Fatal("restored SQL and documents did not reproduce both pre-backup approvals")
		}
	}
	verify(false) // The dedicated target exists but has not been restored.
	restore(west, westArchive)
	verify(true)
	update, err := west.db.ExecContext(drill.ctx, "UPDATE students_records SET version=version+1 WHERE school_id=$1 AND id=$2", "west", fixture.RecordID)
	if err != nil {
		t.Fatal("failed to change a synthetic row on the restored dedicated pool")
	}
	if n, err := update.RowsAffected(); err != nil || n != 1 {
		t.Fatal("corruption drill must change exactly one restored row")
	}
	verify(false)
}
