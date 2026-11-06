package tenantupgrade

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/elcomware/edupilot/platform/database"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/ncruces/go-sqlite3"
)

func TestSQLFenceSQLiteContentionAndRelease(t *testing.T) {
	for _, mode := range []string{"delete", "wal"} {
		t.Run(mode, func(t *testing.T) {
			node := func(name, tenant string) database.Connection {
				return database.Connection{Name: name, Driver: "sqlite", Tenant: tenant, Database: filepath.Join(t.TempDir(), "fixture.sqlite")}
			}
			settings := database.Settings{Control: node("sqlite", ""), Dedicated: []database.Connection{node("tenant_west", "west")}}
			var writers []*sql.DB
			for _, pool := range settings.Nodes() {
				writer, err := sql.Open("sqlite3", pool.Database)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { writer.Close() })
				writer.SetMaxOpenConns(1)
				var actual string
				if err := writer.QueryRow("PRAGMA journal_mode=" + mode).Scan(&actual); err != nil || actual != mode {
					t.Fatal("could not establish fixture journal mode", err)
				}
				if _, err := writer.Exec("PRAGMA busy_timeout=0; CREATE TABLE witness (id INTEGER)"); err != nil {
					t.Fatal(err)
				}
				writers = append(writers, writer)
			}
			var keys []string
			for i := 0; i < 41; i++ {
				keys = append(keys, fmt.Sprintf("fixture%d", i))
			}
			ctx := t.Context()
			err := withSQLFences(ctx, settings, keys, openFenceDB, func(read sourceSnapshots) error {
				for i, pool := range settings.Nodes() {
					if _, err := writers[i].Exec("INSERT INTO witness VALUES (1)"); !errors.Is(err, sqlite3.BUSY) {
						t.Fatalf("independent writer on %s: expected SQLITE_BUSY, got %v", pool.Name, err)
					}
					tx, release, err := read(ctx, pool)
					if err != nil {
						t.Fatal(err)
					}
					var count int
					if err := tx.QueryRow("SELECT COUNT(*) FROM witness").Scan(&count); err != nil || count != 0 {
						t.Fatalf("unexpected fenced contents: %d, %v", count, err)
					}
					release()
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			busy, err := writers[1].BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
			if err != nil {
				t.Fatal("dedicated lock remained after success", err)
			}
			defer busy.Rollback()
			never := func(sourceSnapshots) error { t.Fatal("entered verifier without all locks"); return nil }
			if err := withSQLFences(ctx, settings, keys, openFenceDB, never); !errors.Is(err, errSQLFence) {
				t.Fatal("failed acquisition on second pool accepted", err)
			}
			if _, err := writers[0].Exec("INSERT INTO witness VALUES (2)"); err != nil {
				t.Fatal("failure on second pool leaked first pool lock", err)
			}
			if err := busy.Rollback(); err != nil {
				t.Fatal(err)
			}
			interrupted, cancel := context.WithCancel(ctx)
			defer cancel()
			err = withSQLFences(interrupted, settings, keys, openFenceDB, func(sourceSnapshots) error {
				cancel()
				return interrupted.Err()
			})
			if err == nil {
				t.Fatal("cancelled inspection accepted")
			}
			for _, writer := range writers {
				if _, err := writer.Exec("INSERT INTO witness VALUES (3)"); err != nil {
					t.Fatal("cancellation leaked a source lock", err)
				}
			}
		})
	}
}

// Called with two private, schema-verified PostgreSQL fixtures. NOWAIT makes
// the contention check independent of scheduler timing and CI load.
func exercisePostgresSQLFences(t *testing.T, settings database.Settings, keys []string, open sourceOpener, writers map[string]*sql.DB) {
	t.Helper()
	ctx := t.Context()
	registry, err := sourceRegistry(keys)
	if err != nil {
		t.Fatal(err)
	}
	err = withSQLFences(ctx, settings, keys, open, func(read sourceSnapshots) error {
		for _, node := range settings.Nodes() {
			writer := writers[node.Name]
			for _, table := range registry {
				if table.controlOnly && node.Tenant != "" {
					continue
				}
				// Request the lock that PostgreSQL requires for INSERT, UPDATE
				// and DELETE, from a completely separate connection.
				tx, err := writer.BeginTx(ctx, nil)
				if err != nil {
					t.Fatal(err)
				}
				_, attempt := tx.Exec("LOCK TABLE " + quoteSourceIdentifier(table.name) + " IN ROW EXCLUSIVE MODE NOWAIT")
				tx.Rollback()
				var pgErr *pgconn.PgError
				if !errors.As(attempt, &pgErr) || pgErr.Code != "55P03" {
					t.Fatalf("expected lock_not_available on %s", table.name)
				}
			}
			tx, release, err := read(ctx, node)
			if err != nil {
				t.Fatal(err)
			}
			var owner string
			if err := tx.QueryRow("SELECT owner FROM edu_pool_marker").Scan(&owner); err != nil || owner != node.Owner() {
				t.Fatal("snapshot is not the expected pinned source")
			}
			release()
		}
		source := inventorySource(ctx, settings, keys, SourceSelection{Version: 1, Schools: []string{"east", "west"}}, read)
		if !source.Complete || len(source.Pools) != 2 {
			t.Fatal("could not read the full source inventory through held PostgreSQL locks")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	busy, err := writers["tenant_west"].BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Rollback()
	if _, err := busy.Exec("LOCK TABLE students_records IN ROW EXCLUSIVE MODE NOWAIT"); err != nil {
		t.Fatal("dedicated lock remained after success", err)
	}
	err = withSQLFences(ctx, settings, keys, open, func(sourceSnapshots) error {
		t.Fatal("entered rehearsal with a dedicated legacy writer")
		return nil
	})
	if !errors.Is(err, errSQLFence) {
		t.Fatal("did not refuse active PostgreSQL writer")
	}
	probe, err := writers["postgres"].BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Rollback()
	if _, err := probe.Exec("LOCK TABLE students_records IN ROW EXCLUSIVE MODE NOWAIT"); err != nil {
		t.Fatal("failed rehearsal retained the shared lock", err)
	}
}
