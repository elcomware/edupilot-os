// Package edumigrations maps module-owned SQL plans to Goravel's native migration
// registry. The native migrator owns the control ledger. Dedicated plans open
// transactions on their explicitly named Goravel pool.
package edumigrations

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/elcomware/edupilot-os/app/facades"
	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/elcomware/edupilot-os/platform/outbox"
	"github.com/goravel/framework/contracts/database/schema"
)

type Plan struct {
	ID          string
	ControlOnly bool
	ForwardOnly bool
	Up, Down    []string
	DriverUp    map[string][]string
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
		if statements, exists := p.DriverUp[node.Driver]; exists {
			return statements
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

func (m *registered) Signature() string { return m.plan.ID + "__" + m.node.Name }
func (m *registered) Up() error         { return m.run(false) }
func (m *registered) Down() error       { return m.run(true) }
func (m *registered) run(down bool) error {
	// Goravel v1.18's cached Orm.Connection(name) can report the *previous*
	// connection name even though its query uses the correct physical pool.
	// Keep the native migrator on control. For dedicated DDL, obtain the named
	// physical handle explicitly rather than implementing schema.Connection.
	if m.node.Tenant != "" {
		return m.onDedicated(down)
	}
	frameworkSchema := facades.Schema()
	if frameworkSchema == nil || frameworkSchema.GetConnection() != m.node.Name {
		return fmt.Errorf("%s: incorrect Goravel control connection", m.Signature())
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

func (m *registered) onDedicated(down bool) error {
	orm := facades.Orm()
	if orm == nil || orm.Query() == nil {
		return errors.New("Goravel ORM is unavailable")
	}
	conn := orm.Connection(m.node.Name)
	if conn == nil || conn.Query() == nil {
		return fmt.Errorf("%s: dedicated Goravel connection unavailable", m.Signature())
	}
	db, err := conn.DB()
	if err != nil {
		return fmt.Errorf("%s: resolve named pool: %w", m.Signature(), err)
	}
	pool := database.Pool{DB: db, Driver: m.node.Driver}
	ctx := context.Background() // Goravel Migration.Up/Down do not receive a context.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: start dedicated transaction: %w", m.Signature(), err)
	}
	defer tx.Rollback()
	err = m.apply(down, func(sql string, args ...any) (int64, error) {
		result, err := tx.ExecContext(ctx, pool.Bind(sql), args...)
		if err != nil {
			return 0, err
		}
		return result.RowsAffected()
	})
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Goravel commits the selected connection before updating its default-connection
// history. A local receipt in the DDL transaction closes that crash window for
// arbitrary changes, including ALTER, backfills and plain CREATE/DROP statements.
func (m *registered) apply(down bool, exec func(string, ...any) (int64, error)) error {
	if down && m.plan.ForwardOnly {
		return fmt.Errorf("%s: forward-only migration; recover from a verified snapshot under maintenance", m.Signature())
	}
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
