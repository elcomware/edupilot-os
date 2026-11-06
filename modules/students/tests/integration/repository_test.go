package integration_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/modules/students/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/students/application"
	"github.com/elcomware/edupilot/modules/students/contracts"
	"github.com/elcomware/edupilot/modules/students/domain"
	"github.com/elcomware/edupilot/modules/students/migrations"
	"github.com/elcomware/edupilot/platform/database"
	"github.com/elcomware/edupilot/platform/outbox"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

type fixedCatalog struct{ pool database.Pool }

func (c fixedCatalog) ForTenantContext(context.Context, string) (database.Pool, error) {
	return c.pool, nil
}

func TestRepositoryTransactions(t *testing.T) {
	t.Run("sqlite", func(t *testing.T) {
		db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "test.sqlite"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })
		transactionCases(t, database.Pool{DB: db, Driver: "sqlite"})
	})
	t.Run("postgres", func(t *testing.T) {
		dsn := os.Getenv("EDUPILOT_TEST_POSTGRES_DSN")
		if dsn == "" {
			t.Skip("set EDUPILOT_TEST_POSTGRES_DSN to a disposable test database")
		}
		cfg, err := pgx.ParseConfig(dsn)
		if err != nil {
			t.Fatal(err)
		}
		var token [12]byte
		if _, err = rand.Read(token[:]); err != nil {
			t.Fatal(err)
		}
		schema := "students_test_" + hex.EncodeToString(token[:])
		cfg.RuntimeParams["search_path"] = schema
		db := stdlib.OpenDB(*cfg)
		t.Cleanup(func() { _ = db.Close() })
		if _, err = db.Exec("CREATE SCHEMA " + schema); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if _, err := db.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
				t.Error(err)
			}
		})
		transactionCases(t, database.Pool{DB: db, Driver: "postgres"})
	})
}

func transactionCases(t *testing.T, p database.Pool) {
	t.Helper()
	ctx := context.Background()
	apply := func(statements []string) {
		t.Helper()
		for _, stmt := range statements {
			if _, err := p.DB.Exec(stmt); err != nil {
				t.Fatal(err)
			}
		}
	}
	apply(migrations.SchemaUp)
	repo := persistence.New(fixedCatalog{p})
	svc := application.New(repo)
	// This simulated incomplete deployment forces Append to fail after INSERT.
	if _, err := svc.Register(ctx, "north", "actor", "Avery", "Lee"); err == nil {
		t.Fatal("mutation must fail when event cannot be recorded")
	}
	page, err := svc.List(ctx, "north", 25, 0)
	if err != nil || page.Total != 0 {
		t.Fatalf("mutation escaped rollback: %+v %v", page, err)
	}
	apply(outbox.SchemaUp)
	first, err := svc.Register(ctx, "north", "actor", "Avery", "Lee")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Register(ctx, "south", "actor", "Rae", "Lopez")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Lookup(ctx, "north", second.ID); !errors.Is(err, contracts.ErrNotFound) {
		t.Fatalf("other school leaked: %v", err)
	}
	page, err = svc.List(ctx, "north", 25, 100)
	if err != nil || page.Total != 1 || len(page.Items) != 0 {
		t.Fatalf("empty page should retain scoped total: %+v %v", page, err)
	}
	old, err := repo.Get(ctx, "north", first.ID)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := old.Rename(1, "Ava", "Lee")
	if err != nil {
		t.Fatal(err)
	}
	competing, err := old.Archive(1)
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.Replace(ctx, changed, 1, "editor"); err != nil {
		t.Fatal(err)
	}
	if err = repo.Replace(ctx, competing, 1, "editor2"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("stale write applied: %v", err)
	}
	if err = svc.Archive(ctx, "north", "actor", first.ID, 2); err != nil {
		t.Fatal(err)
	}
	page, err = svc.List(ctx, "north", 25, 0)
	if err != nil || page.Total != 0 {
		t.Fatalf("archive still active: %+v %v", page, err)
	}
	got, err := svc.Lookup(ctx, "north", first.ID)
	if err != nil || got.Status != "archived" || got.Version != 3 {
		t.Fatalf("unexpected archived record: %+v %v", got, err)
	}

	// An event for every committed version, none for the competing stale write.
	rows, err := p.DB.QueryContext(ctx, p.Bind(`SELECT event_type,payload FROM edu_outbox WHERE tenant_id=? AND aggregate_id=?`), "north", first.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	events := map[int64]string{}
	for rows.Next() {
		var kind, raw string
		if err = rows.Scan(&kind, &raw); err != nil {
			t.Fatal(err)
		}
		var event struct {
			StudentID string `json:"studentId"`
			Version   int64  `json:"version"`
			ActorID   string `json:"actorId"`
		}
		if err = json.Unmarshal([]byte(raw), &event); err != nil {
			t.Fatal(err)
		}
		if event.StudentID != first.ID || event.ActorID == "" || strings.Contains(raw, "Lee") {
			t.Fatalf("wrong event payload: %s", raw)
		}
		if _, exists := events[event.Version]; exists {
			t.Fatalf("duplicate event for version %d", event.Version)
		}
		events[event.Version] = kind
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[1] != "students.registered.v1" || events[2] != "students.updated.v1" || events[3] != "students.archived.v1" {
		t.Fatalf("events: %v", events)
	}
}
