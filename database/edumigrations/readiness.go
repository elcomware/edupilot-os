package edumigrations

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/elcomware/edupilot-os/app/facades"
	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/database/migration"
)

var identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func historyTable() (string, error) {
	table := facades.Config().GetString("database.migrations.table", "migrations")
	if !identifier.MatchString(table) {
		return "", errors.New("database.migrations.table must be a simple SQL identifier")
	}
	return table, nil
}

// Run invokes the native migrator directly so its real error reaches the deployment process.
func Run(ctx context.Context, catalog *database.Catalog) error {
	s := facades.Schema()
	if s == nil || len(s.Migrations()) == 0 {
		return errors.New("no Goravel schema migrations registered")
	}
	table, err := historyTable()
	if err != nil {
		return err
	}
	if err := migration.NewMigrator(facades.Artisan(), s, table).Run(); err != nil {
		return fmt.Errorf("apply Goravel migrations: %w", err)
	}
	return CheckReady(ctx, catalog, s.Migrations())
}

// CheckReady checks the default Goravel ledger for *every* registered migration,
// local DDL receipts for every EduPilot plan, and ownership of every configured DB.
// Reading the complete ledger and one receipt set per pool bounds query count by pools.
func CheckReady(ctx context.Context, catalog *database.Catalog, expected []schema.Migration) error {
	if catalog == nil || catalog.Control.DB == nil || len(expected) == 0 {
		return errors.New("database catalog or registered migrations unavailable")
	}
	if err := catalog.CheckReady(ctx); err != nil {
		return err
	}
	table, err := historyTable()
	if err != nil {
		return err
	}
	history, err := signatures(ctx, catalog.Control, `SELECT migration FROM "`+table+`"`)
	if err != nil {
		return fmt.Errorf("read Goravel migration ledger: %w", err)
	}

	pools := catalog.Named()
	receipts := make(map[string]map[string]int, len(pools))
	placements := make(map[string]bool, len(pools))
	for _, m := range expected {
		if m == nil || m.Signature() == "" {
			return errors.New("invalid registered migration")
		}
		id := m.Signature()
		if history[id] != 1 {
			return fmt.Errorf("expected one Goravel ledger entry for %s, found %d", id, history[id])
		}
		plan, isLocal := m.(*registered)
		if !isLocal {
			continue // Native upstream migrations use only the native ledger.
		}
		name := plan.node.Name
		p, exists := pools[name]
		if !exists {
			return fmt.Errorf("migration %s has no configured pool %s", id, name)
		}
		if receipts[name] == nil {
			receipts[name], err = signatures(ctx, p, "SELECT signature FROM edu_migration_receipts")
			if err != nil {
				return fmt.Errorf("read %s migration receipts: %w", name, err)
			}
		}
		if receipts[name][id] != 1 {
			return fmt.Errorf("missing local migration receipt for %s on %s", id, name)
		}
		if plan.plan.placement {
			placements[name] = true
		}
	}
	for name := range pools {
		if !placements[name] {
			return fmt.Errorf("missing registered placement migration for %s", name)
		}
	}
	return nil
}

func signatures(ctx context.Context, pool database.Pool, sql string) (map[string]int, error) {
	rows, err := pool.DB.QueryContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]int)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id]++
	}
	return result, rows.Err()
}
