package edumigrations_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/platform/database"
	"github.com/elcomware/edupilot/platform/outbox"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func sqlitePool(t *testing.T, file string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+file+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// Never run DDL in public or a supplied existing schema. A failed CREATE must not
// arm a cleanup for someone else's tables. The CI role only needs CREATE on this DB.
func postgresPool(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin := stdlib.OpenDB(*config)
	t.Cleanup(func() { admin.Close() })
	var id [8]byte
	if _, err := rand.Read(id[:]); err != nil {
		t.Fatal(err)
	}
	schema := "edu_platform_" + hex.EncodeToString(id[:])
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Error(err)
		}
	})
	config.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*config)
	t.Cleanup(func() { db.Close() })
	return db
}

func apply(t *testing.T, conn database.Connection, pool database.Pool) {
	t.Helper()
	for _, plan := range []edumigrations.Plan{edumigrations.PlacementPlan(), edumigrations.OutboxPlan()} {
		tx, err := pool.DB.Begin()
		if err != nil {
			t.Fatal(err)
		}
		for _, stmt := range plan.Statements(conn, false) {
			if _, err := tx.Exec(stmt); err != nil {
				tx.Rollback()
				t.Fatalf("%s: %v", plan.ID, err)
			}
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
}

func count(t *testing.T, p database.Pool, query string, want int) {
	t.Helper()
	var got int
	if err := p.DB.QueryRow(query).Scan(&got); err != nil || got != want {
		t.Fatalf("count %q = %d, want %d: %v", query, got, want, err)
	}
}

func enqueue(t *testing.T, p database.Pool, school, eventID string, commit bool) {
	t.Helper()
	tx, err := p.DB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	event := outbox.Event{ID: eventID, TenantID: school, Type: "students.registered.v1",
		AggregateID: "student-1", Payload: []byte(`{"student_id":"student-1"}`)}
	if err := outbox.Append(context.Background(), tx, p, event); err != nil {
		t.Fatal(err)
	}
	if commit {
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
}

type failingConsumer struct{}

func (failingConsumer) Consume(ctx context.Context, tx *sql.Tx, pool database.Pool, e outbox.Event) error {
	// A consumer that fails *after* an effect must roll it back along with the acknowledgement.
	if err := (outbox.AuditInbox{}).Consume(ctx, tx, pool, e); err != nil {
		return err
	}
	return errors.New("injected failure")
}

func TestHybridRoutingAndTransactionalDelivery(t *testing.T) {
	for _, driver := range []string{"sqlite", "postgres"} {
		t.Run(driver, func(t *testing.T) {
			pgDSN := os.Getenv("EDUPILOT_TEST_POSTGRES_DSN")
			if driver == "postgres" && pgDSN == "" {
				t.Skip("set EDUPILOT_TEST_POSTGRES_DSN for a PostgreSQL control/shared pool")
			}
			root := t.TempDir()
			shared := database.Connection{Name: driver, Driver: driver, Database: filepath.Join(root, "shared.sqlite")}
			if driver == "postgres" {
				// The injected resolver below supplies our isolated test schema.
				shared.Host, shared.Port, shared.Username, shared.Database, shared.SSLMode = "localhost", 5432, "test", "test", "disable"
			}
			west := database.Connection{Name: "tenant_west", Tenant: "west", Driver: "sqlite", Database: filepath.Join(root, "west.sqlite")}
			s := database.Settings{Control: shared, Dedicated: []database.Connection{west}}
			dbs := map[string]*sql.DB{west.Name: sqlitePool(t, west.Database)}
			if driver == "postgres" {
				dbs[shared.Name] = postgresPool(t, pgDSN)
			} else {
				dbs[shared.Name] = sqlitePool(t, shared.Database)
			}
			resolve := func(n database.Connection) (*sql.DB, error) { return dbs[n.Name], nil }
			catalog, err := database.Open(s, resolve)
			if err != nil {
				t.Fatal(err)
			}
			for i, pool := range catalog.All() {
				apply(t, s.Nodes()[i], pool)
			}
			ctx := context.Background()
			if err := catalog.CheckReady(ctx); err != nil {
				t.Fatal(err)
			}
			eastPool, err := catalog.ForTenantContext(ctx, "east")
			if err != nil || eastPool.DB != catalog.Control.DB {
				t.Fatalf("shared placement: %v", err)
			}
			westPool, err := catalog.ForTenantContext(ctx, "west")
			if err != nil || westPool.DB == catalog.Control.DB {
				t.Fatalf("dedicated placement: %v", err)
			}
			missingOverride, err := database.Open(database.Settings{Control: shared}, resolve)
			if err != nil {
				t.Fatal(err)
			}
			if err := missingOverride.CheckReady(ctx); !errors.Is(err, database.ErrPlacement) {
				t.Fatalf("readiness accepted removed dedicated override: %v", err)
			}
			if _, err := missingOverride.ForTenantContext(ctx, "west"); !errors.Is(err, database.ErrPlacement) {
				t.Fatalf("removed config routed formerly dedicated school: %v", err)
			}

			enqueue(t, eastPool, "east", "rollback", false)
			count(t, eastPool, "SELECT count(*) FROM edu_outbox", 0)
			enqueue(t, eastPool, "east", "committed-east", true)
			enqueue(t, westPool, "west", "committed-west", true)
			clock := time.Now().UTC().Add(time.Second)
			dispatch := outbox.Dispatcher{Now: func() time.Time { return clock }, Consumer: failingConsumer{}}
			if found, err := dispatch.Step(ctx, eastPool); !found || err == nil {
				t.Fatalf("failed attempt: found=%v, err=%v", found, err)
			}
			count(t, eastPool, "SELECT count(*) FROM edu_audit_inbox", 0)
			count(t, eastPool, "SELECT count(*) FROM edu_outbox WHERE published_at IS NULL AND attempts = 1", 1)
			dispatch.Consumer = outbox.AuditInbox{}
			if found, err := dispatch.Step(ctx, eastPool); found || err != nil {
				t.Fatalf("retry before backoff: found=%v, err=%v", found, err)
			}
			clock = clock.Add(2 * time.Second)
			for _, p := range catalog.All() {
				if found, err := dispatch.Step(ctx, p); !found || err != nil {
					t.Fatalf("deliver: found=%v, err=%v", found, err)
				}
				count(t, p, "SELECT count(*) FROM edu_audit_inbox", 1)
			}
			count(t, eastPool, "SELECT count(*) FROM edu_audit_inbox WHERE tenant_id = 'west'", 0)
			count(t, westPool, "SELECT count(*) FROM edu_audit_inbox WHERE tenant_id = 'east'", 0)

			// Simulate replay and an abandoned lease. Once the lease expires,
			// redelivery sees the existing audit row and acknowledges exactly once.
			if _, err := eastPool.DB.Exec(eastPool.Bind("UPDATE edu_outbox SET published_at = NULL, lease_id = 'old', lease_until = ?"), clock.Add(time.Hour).UnixMilli()); err != nil {
				t.Fatal(err)
			}
			if found, err := dispatch.Step(ctx, eastPool); found || err != nil {
				t.Fatalf("stole active lease: %v %v", found, err)
			}
			clock = clock.Add(time.Hour)
			if found, err := dispatch.Step(ctx, eastPool); !found || err != nil {
				t.Fatalf("expired lease: %v %v", found, err)
			}
			count(t, eastPool, "SELECT count(*) FROM edu_audit_inbox", 1)
			count(t, eastPool, "SELECT count(*) FROM edu_outbox WHERE published_at IS NOT NULL AND attempts = 3", 1)

			enqueue(t, eastPool, "east", "poison", true)
			dispatch.Consumer = failingConsumer{}
			for i := 0; i < 8; i++ {
				clock = clock.Add(time.Hour)
				if found, err := dispatch.Step(ctx, eastPool); !found || err == nil {
					t.Fatalf("expected retry attempt %d: %v %v", i+1, found, err)
				}
			}
			clock = clock.Add(time.Hour)
			if found, err := dispatch.Step(ctx, eastPool); found || err != nil {
				t.Fatalf("poison event still visible: %v %v", found, err)
			}
			count(t, eastPool, "SELECT count(*) FROM edu_outbox WHERE dead_at IS NOT NULL AND attempts = 8", 1)
			count(t, eastPool, "SELECT count(*) FROM edu_audit_inbox", 1)
		})
	}
}
