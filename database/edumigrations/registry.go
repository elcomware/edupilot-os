// Package edumigrations registers module-owned Goravel migrations on each configured pool.
package edumigrations

import (
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/migration"
	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/database/schema"
)

const receiptTable = "edu_foundation_receipts"

type registered struct {
	node     database.Connection
	key      string
	create   func(migration.Schema) error
	baseline bool
}

func OnPool(node database.Connection, key string, create func(migration.Schema) error) schema.Migration {
	return &registered{node: node, key: key, create: create}
}
func PoolBaseline(node database.Connection, create func(migration.Schema) error) schema.Migration {
	return &registered{node: node, key: "01_pool", create: create, baseline: true}
}
func (m *registered) Signature() string { return "20261106_foundation_" + m.key + "__" + m.node.Name }
func (m *registered) Down() error {
	return errors.New("pool migrations require restoration from a verified backup")
}
func (m *registered) Up() error {
	// Goravel already wraps the control connection in a transaction. A dedicated
	// database needs its own transaction and receipt before the control ledger advances.
	db := facades.Schema()
	if m.node.Tenant == "" {
		return m.apply(db)
	}
	target := db.Connection(m.node.Name)
	return target.Orm().Transaction(func(tx orm.Query) error {
		original := target.Orm().Query()
		target.Orm().SetQuery(tx)
		defer target.Orm().SetQuery(original)
		return m.apply(target)
	})
}
func (m *registered) apply(db schema.Schema) error {
	q := db.Orm().Query()
	// Inspect the selected connection's schema, including PostgreSQL search_path.
	exists, err := tableExists(q, m.node.Driver, receiptTable)
	if err != nil {
		return err
	}
	if !exists {
		if !m.baseline {
			return errors.New("missing pool baseline")
		}
		// Artisan can invoke Up directly. Accept only an empty target (and
		// the empty native ledger on control) before claiming the pool.
		objects := applicationRelations(m.node.Driver)
		tables := q.Table(objects.catalog).Where(objects.filter)
		if m.node.Tenant == "" {
			ledger, err := historyTable()
			if err != nil {
				return err
			}
			occupied, err := q.Table(ledger).Exists()
			if err != nil {
				return err
			}
			if occupied {
				return errors.New("unrecognized migration history")
			}
			tables = tables.Where(objects.name+" <> ?", ledger)
		}
		occupied, err := tables.Exists()
		if err != nil {
			return err
		}
		if occupied {
			return errors.New("select an empty database for the pool baseline")
		}
		if err := db.Create(receiptTable, func(t schema.Blueprint) { t.String("signature"); t.Primary("signature") }); err != nil {
			return err
		}
	}
	recorded, err := q.Table(receiptTable).Where("signature", m.Signature()).Exists()
	if err != nil || recorded {
		return err
	}
	if err = m.create(db); err != nil {
		return fmt.Errorf("%s: %w", m.Signature(), err)
	}
	if m.baseline {
		if err = q.Table("edu_pool_marker").Create(map[string]any{"singleton": 1, "owner": m.node.Owner(), "pool_id": rand.Text()}); err != nil {
			return err
		}
	}
	return q.Table(receiptTable).Create(map[string]any{"signature": m.Signature()})
}
func tableExists(q orm.Query, driver, name string) (bool, error) {
	if driver == "postgres" {
		return q.Table("information_schema.tables").Where("table_schema=current_schema() AND table_type='BASE TABLE'").Where("table_name", name).Exists()
	}
	return q.Table("sqlite_master").Where("type", "table").Where("name", name).Exists()
}

// Both the installer preflight and the native migration inspect the selected schema.
// PostgreSQL creates dependent indexes and sequences for the native ledger itself.
type relationInventory struct{ catalog, name, filter string }

func applicationRelations(driver string) relationInventory {
	if driver == "postgres" {
		return relationInventory{
			catalog: "pg_catalog.pg_class",
			name:    "relname",
			filter: `relnamespace = (SELECT oid FROM pg_catalog.pg_namespace WHERE nspname = current_schema())
				AND (relkind IN ('r', 'p', 'v', 'm', 'f') OR (relkind = 'S' AND NOT EXISTS (
					SELECT 1 FROM pg_catalog.pg_depend
					WHERE classid = 'pg_catalog.pg_class'::regclass AND objid = pg_class.oid AND deptype IN ('a', 'i')
				)))`,
		}
	}
	return relationInventory{catalog: "sqlite_master", name: "name", filter: "name NOT GLOB 'sqlite_*'"}
}
