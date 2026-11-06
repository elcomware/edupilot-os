package edumigrations

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/modules/core/database"
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
	if catalog == nil || catalog.Control.DB == nil {
		return errors.New("database catalog unavailable")
	}
	s := facades.Schema()
	if s == nil || len(s.Migrations()) == 0 {
		return errors.New("no Goravel schema migrations registered")
	}
	table, err := historyTable()
	if err != nil {
		return err
	}
	// Inspect every selected target before creating any tables. Refuse a nonempty
	// unrecognized database, including a dedicated target on a fresh control.
	pools := catalog.Named()
	identities := make(map[string]string, len(pools))
	for _, step := range s.Migrations() {
		if m, ok := step.(*registered); ok && m.baseline {
			pool, exists := pools[m.node.Name]
			if !exists {
				return fmt.Errorf("configured pool %s unavailable", m.node.Name)
			}
			id, err := checkBaseline(ctx, pool, table, m)
			if err != nil {
				return fmt.Errorf("inspect pool %s: %w", m.node.Name, err)
			}
			if id != "" {
				identities[m.node.Owner()] = id
			}
		}
	}
	if err := database.ValidatePoolIdentities(identities); err != nil {
		return err
	}
	// Goravel's PostgreSQL HasTable implementation enumerates every schema and
	// then compares only the unqualified table name. A migrations table in an
	// unrelated schema can therefore suppress creation in the selected schema.
	// Resolve and, when necessary, create the repository against the verified
	// control handle before handing execution to the native migrator.
	if err := ensureHistoryRepository(ctx, catalog.Control, s, table); err != nil {
		return err
	}
	if err := migration.NewMigrator(facades.Artisan(), s, table).Run(); err != nil {
		return fmt.Errorf("apply Goravel migrations: %w", err)
	}
	return CheckReady(ctx, catalog, s.Migrations())
}

func ensureHistoryRepository(ctx context.Context, control database.Pool, s schema.Schema, table string) error {
	exists, err := historyRepositoryExists(ctx, control, table)
	if err != nil {
		return fmt.Errorf("inspect Goravel migration ledger: %w", err)
	}
	if exists {
		return nil
	}

	createErr := migration.NewRepository(s, table).CreateRepository()
	exists, inspectErr := historyRepositoryExists(ctx, control, table)
	if inspectErr != nil {
		return fmt.Errorf("verify Goravel migration ledger: %w", inspectErr)
	}
	if exists {
		// A concurrent native migrator may win table creation. The exact-schema
		// recheck makes that race safe without accepting another schema's table.
		return nil
	}
	if createErr != nil {
		return fmt.Errorf("create Goravel migration ledger: %w", createErr)
	}
	return errors.New("Goravel migration ledger was not created in the control schema")
}

func historyRepositoryExists(ctx context.Context, control database.Pool, table string) (bool, error) {
	query := "SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?"
	if control.Driver == "postgres" {
		query = "SELECT count(*) FROM information_schema.tables WHERE table_schema=current_schema() AND table_type='BASE TABLE' AND table_name=?"
	}
	var count int
	if err := control.DB.QueryRowContext(ctx, control.Bind(query), table).Scan(&count); err != nil {
		return false, err
	}
	return count == 1, nil
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
			receipts[name], err = signatures(ctx, p, "SELECT signature FROM "+receiptTable)
			if err != nil {
				return fmt.Errorf("read %s migration receipts: %w", name, err)
			}
		}
		if receipts[name][id] != 1 {
			return fmt.Errorf("missing local migration receipt for %s on %s", id, name)
		}
		if plan.baseline {
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

func checkBaseline(ctx context.Context, p database.Pool, ledger string, baseline *registered) (string, error) {
	found, err := historyRepositoryExists(ctx, p, receiptTable)
	if err != nil {
		return "", err
	}
	if found {
		var owner, poolID string
		err = p.DB.QueryRowContext(ctx, "SELECT owner, pool_id FROM edu_pool_marker WHERE singleton=1").Scan(&owner, &poolID)
		if err != nil {
			return "", fmt.Errorf("read pool marker: %w", err)
		}
		if owner != baseline.node.Owner() || poolID == "" {
			return "", database.ErrPlacement
		}
		var recorded int
		err = p.DB.QueryRowContext(ctx, p.Bind("SELECT count(*) FROM "+receiptTable+" WHERE signature=?"), baseline.Signature()).Scan(&recorded)
		if err != nil {
			return "", err
		}
		if recorded != 1 {
			return "", errors.New("missing pool baseline receipt")
		}
		return poolID, nil
	}
	objects := applicationRelations(p.Driver)
	query := "SELECT count(*) FROM " + objects.catalog + " WHERE " + objects.filter
	var n int
	if err := p.DB.QueryRowContext(ctx, query).Scan(&n); err != nil {
		return "", err
	}
	if n == 0 {
		return "", nil
	}
	// A crash before the first migration may leave an empty native history table.
	if n == 1 && baseline.node.Tenant == "" {
		exists, err := historyRepositoryExists(ctx, p, ledger)
		if err != nil {
			return "", err
		}
		if exists {
			var count int
			if err := p.DB.QueryRowContext(ctx, `SELECT count(*) FROM "`+ledger+`"`).Scan(&count); err != nil {
				return "", err
			}
			if count == 0 {
				return "", nil
			}
		}
	}
	return "", errors.New("unrecognized database schema; select a new empty database for this installation")
}
