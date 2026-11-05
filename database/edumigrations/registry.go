// Package edumigrations maps module-owned SQL plans to Goravel's native migration
// registry. Goravel opens the transaction on Connection() and stores each unique
// signature in the control database's normal migration ledger.
package edumigrations

import (
	"crypto/rand"
	"fmt"

	"github.com/elcomware/edupilot-os/app/facades"
	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/elcomware/edupilot-os/platform/outbox"
	"github.com/goravel/framework/contracts/database/schema"
)

type Plan struct {
	ID          string
	ControlOnly bool
	Up, Down    []string
	placement   bool
}

func PlacementPlan() Plan {
	return Plan{ID: "20260101000001_edu_placement", placement: true}
}

func OutboxPlan() Plan {
	return Plan{ID: "20260101000002_edu_outbox", Up: outbox.SchemaUp, Down: outbox.SchemaDown}
}

// Statements exposes the exact DDL for integration fixtures. In production only
// Goravel's migrator should execute it, on its own connection and transaction.
func (p Plan) Statements(node database.Connection, down bool) []string {
	if !p.placement {
		if down {
			return p.Down
		}
		return p.Up
	}
	if down {
		statements := []string{"DROP TABLE IF EXISTS edu_pool_marker"}
		if node.Tenant == "" {
			statements = append(statements, "DROP TABLE IF EXISTS edu_tenant_placement")
		}
		return statements
	}

	// Owner is either "shared" or a validated tenant slug. The NOT NULL
	// constraint forces the transaction to fail if two names resolve to one DB.
	up := []string{
		`CREATE TABLE IF NOT EXISTS edu_pool_marker (
			singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
			owner TEXT NOT NULL, pool_id TEXT NOT NULL
		)`,
		fmt.Sprintf(`INSERT INTO edu_pool_marker (singleton, owner, pool_id) VALUES (1, '%s', '%s')
			ON CONFLICT (singleton) DO UPDATE SET owner =
				CASE WHEN edu_pool_marker.owner = excluded.owner THEN edu_pool_marker.owner ELSE NULL END`,
			node.Owner(), rand.Text()),
	}
	if node.Tenant == "" {
		up = append(up, `CREATE TABLE IF NOT EXISTS edu_tenant_placement (
			tenant_id TEXT PRIMARY KEY NOT NULL, pool_id TEXT NOT NULL
		)`)
	}
	return up
}

type registered struct {
	plan Plan
	node database.Connection
}

var _ schema.Migration = (*registered)(nil)
var _ schema.Connection = (*registered)(nil)

func (m *registered) Signature() string  { return m.plan.ID + "__" + m.node.Name }
func (m *registered) Connection() string { return m.node.Name }
func (m *registered) Up() error          { return m.run(false) }
func (m *registered) Down() error        { return m.run(true) }
func (m *registered) run(down bool) error {
	frameworkSchema := facades.Schema()
	if frameworkSchema == nil || frameworkSchema.GetConnection() != m.node.Name {
		return fmt.Errorf("%s: incorrect Goravel schema connection", m.Signature())
	}
	tx := frameworkSchema.Orm().Query()
	if tx == nil || !tx.InTransaction() {
		return fmt.Errorf("%s: Goravel migration transaction required", m.Signature())
	}
	return m.apply(down, func(sql string, args ...any) (int64, error) {
		result, err := tx.Exec(sql, args...)
		if err != nil {
			return 0, err
		}
		return result.RowsAffected, nil
	})
}

// Goravel commits the selected connection before updating its default-connection
// history. A local receipt in the DDL transaction closes that crash window for
// arbitrary changes, including ALTER, backfills and plain CREATE/DROP statements.
func (m *registered) apply(down bool, exec func(string, ...any) (int64, error)) error {
	const receipts = `CREATE TABLE IF NOT EXISTS edu_migration_receipts
		(signature TEXT PRIMARY KEY NOT NULL)`
	if _, err := exec(receipts); err != nil {
		return fmt.Errorf("%s: initialize migration receipt: %w", m.Signature(), err)
	}
	mark := "INSERT INTO edu_migration_receipts (signature) VALUES (?) ON CONFLICT (signature) DO NOTHING"
	if down {
		mark = "DELETE FROM edu_migration_receipts WHERE signature = ?"
	}
	changed, err := exec(mark, m.Signature())
	if err != nil {
		return fmt.Errorf("%s: record migration receipt: %w", m.Signature(), err)
	}
	if changed == 0 {
		return nil
	}
	for _, stmt := range m.plan.Statements(m.node, down) {
		if _, err := exec(stmt); err != nil {
			return fmt.Errorf("%s: %w", m.Signature(), err)
		}
	}
	return nil
}

// Register is used by bootstrap.Migrations. Each dedicated DB receives its own
// signatures. Adding a new DB consequently applies all applicable plans to it.
func Register(settings database.Settings, plans ...Plan) []schema.Migration {
	if err := settings.Validate(); err != nil {
		panic(err)
	}
	seen := make(map[string]bool)
	var registeredMigrations []schema.Migration
	for _, plan := range plans {
		if plan.ID == "" || seen[plan.ID] {
			panic("missing or duplicate EduPilot migration ID: " + plan.ID)
		}
		seen[plan.ID] = true
		for _, node := range settings.Nodes() {
			if plan.ControlOnly && node.Tenant != "" {
				continue
			}
			registeredMigrations = append(registeredMigrations, &registered{plan, node})
		}
	}
	return registeredMigrations
}
