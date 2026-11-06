package tenantupgrade

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/elcomware/edupilot/platform/database"
	"github.com/jackc/pgx/v5"
)

// SQLFenceReport describes a completed exercise. It cannot be reused as a
// maintenance permit: the SQL locks have already been released at return.
type SQLFenceReport struct {
	Version        int           `json:"version"`
	Classification string        `json:"classification"`
	Complete       bool          `json:"complete"`
	Pools          int           `json:"pools"`
	ManifestSHA256 string        `json:"manifestSha256,omitempty"`
	SourceSHA256   string        `json:"sourceSha256,omitempty"`
	Issues         []DryRunIssue `json:"issues"`
}

var errSQLFence = errors.New("source SQL fence unavailable")

// RehearseSQLFence verifies the approved source on the same transactions
// holding its write locks. All handles roll back; no copy phase is exposed.
func (u Upgrade) RehearseSQLFence(ctx context.Context) (SQLFenceReport, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	report := SQLFenceReport{Version: 1, Classification: "legacy-sql-fence-rehearsal", Issues: []DryRunIssue{}}
	if err := ctx.Err(); err != nil {
		return SQLFenceReport{}, err
	}
	if u.manifest.Validate() != nil || u.selection.Version != 2 || u.selection.Validate() != nil {
		report.Issues = append(report.Issues, DryRunIssue{Code: "invalid_fence_inputs"})
		return report, nil
	}
	var evidence DryRunReport
	err := withSQLFences(ctx, u.settings, u.workspaceKeys, openFenceDB, func(read sourceSnapshots) error {
		var err error
		evidence, err = u.withSnapshots(read).DryRun(ctx)
		return err
	})
	if ctx.Err() != nil {
		return SQLFenceReport{}, ctx.Err()
	}
	if err != nil {
		report.Issues = append(report.Issues, DryRunIssue{Code: "sql_fence_unavailable"})
		return report, nil
	}
	report.Pools, report.Issues = evidence.Pools, evidence.Issues
	if evidence.Complete && evidence.ReadyForPrepare {
		report.Complete = true
		report.ManifestSHA256, report.SourceSHA256 = evidence.ManifestSHA256, evidence.SourceSHA256
	}
	return report, nil
}

// Acquire every lock before calling inspect. Readers borrow the pinned
// transactions so a reconnect cannot silently substitute an unfenced source.
func withSQLFences(ctx context.Context, settings database.Settings, keys []string, open sourceOpener, inspect func(sourceSnapshots) error) (err error) {
	nodes := settings.Nodes()
	tables, err := sourceRegistry(keys)
	if err != nil || len(nodes) > 128 || settings.Validate() != nil {
		return errSQLFence
	}
	slices.SortFunc(nodes, func(a, b database.Connection) int { return strings.Compare(a.Name, b.Name) })
	held := make(map[string]*sql.Tx, len(nodes))
	for _, node := range nodes {
		db, openErr := open(ctx, node)
		if openErr != nil {
			return errSQLFence
		}
		defer db.Close()
		// Tx.Rollback can return ErrTxDone while database/sql is still
		// rolling back on cancellation. Closing the pinned connection waits
		// for that rollback before a caller can immediately retry.
		conn, connErr := db.Conn(ctx)
		if connErr != nil {
			return errSQLFence
		}
		defer conn.Close()
		opts := &sql.TxOptions{}
		if node.Driver == "sqlite" {
			// The native SQLite driver maps this to BEGIN IMMEDIATE, reserving
			// the sole writer in both rollback-journal and WAL modes.
			opts.Isolation = sql.LevelSerializable
		}
		tx, beginErr := conn.BeginTx(ctx, opts)
		if beginErr != nil {
			return errSQLFence
		}
		defer func() {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = errSQLFence
			}
		}()
		if node.Driver == "postgres" {
			if err := lockRegisteredTables(ctx, tx, node, tables); err != nil {
				return errSQLFence
			}
		}
		held[node.Name] = tx
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return inspect(func(_ context.Context, node database.Connection) (*sql.Tx, func(), error) {
		tx := held[node.Name]
		if tx == nil {
			return nil, nil, errSQLFence
		}
		return tx, func() {}, nil
	})
}

func lockRegisteredTables(ctx context.Context, tx *sql.Tx, node database.Connection, tables []sourceTable) error {
	// Qualify the relation names. Do not search a fallback schema for a
	// missing legacy table. Production handles use the native public schema.
	var schema string
	if err := tx.QueryRowContext(ctx, "SELECT current_schema()").Scan(&schema); err != nil || schema == "" {
		return errSQLFence
	}
	names := make([]string, 0, len(tables))
	for _, table := range tables {
		if table.controlOnly && node.Tenant != "" {
			continue
		}
		names = append(names, pgx.Identifier{schema, table.name}.Sanitize())
	}
	_, err := tx.ExecContext(ctx, "LOCK TABLE "+strings.Join(names, ",")+" IN EXCLUSIVE MODE NOWAIT")
	return err
}
