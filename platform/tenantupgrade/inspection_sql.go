package tenantupgrade

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func openInspectionDB(ctx context.Context, node database.Connection) (*sql.DB, error) {
	var db *sql.DB
	if node.Driver == "sqlite" {
		info, err := os.Stat(node.Database)
		if err != nil || !info.Mode().IsRegular() {
			return nil, errors.New("source unavailable")
		}
		path, err := filepath.Abs(node.Database)
		if err != nil {
			return nil, errors.New("source unavailable")
		}
		uri := url.URL{Scheme: "file", Path: path, RawQuery: url.Values{"mode": {"ro"}, "_pragma": {"query_only(1)", "busy_timeout(5000)"}}.Encode()}
		db, err = sql.Open("sqlite3", uri.String())
		if err != nil {
			return nil, errors.New("source unavailable")
		}
	} else {
		uri := url.URL{Scheme: "postgres", User: url.UserPassword(node.Username, node.Password), Host: net.JoinHostPort(node.Host, strconv.Itoa(node.Port)), Path: "/" + node.Database}
		uri.RawQuery = url.Values{"sslmode": {node.SSLMode}, "search_path": {"public"}, "timezone": {"UTC"}, "connect_timeout": {"5"}, "default_transaction_read_only": {"on"}}.Encode()
		config, err := pgx.ParseConfig(uri.String())
		if err != nil {
			return nil, errors.New("source unavailable")
		}
		db = stdlib.OpenDB(*config)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, errors.New("source unavailable")
	}
	return db, nil
}
func inspectionTables(ctx context.Context, tx *sql.Tx, driver string) (map[string]bool, error) {
	query := "SELECT " + boundedInspectionText("name", 65, driver) + " FROM sqlite_master WHERE type='table' LIMIT 8193"
	if driver == "postgres" {
		query = "SELECT substr(table_name,1,65) FROM information_schema.tables WHERE table_schema=current_schema() AND table_type='BASE TABLE' LIMIT 8193"
	}
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	available := map[string]bool{}
	count := 0
	for rows.Next() {
		count++
		if count > 8192 {
			return nil, errors.New("source catalog exceeds inspection limit")
		}
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		available[name] = true
	}
	return available, rows.Err()
}

func beginInspectionSnapshot(ctx context.Context, db *sql.DB, driver string) (*sql.Tx, error) {
	// SQLite serializable maps to BEGIN IMMEDIATE and requests a writer lock.
	// Deferred read transactions retain a snapshot without requesting a writer.
	isolation := sql.LevelDefault
	if driver == "postgres" {
		isolation = sql.LevelRepeatableRead
	}
	return db.BeginTx(ctx, &sql.TxOptions{Isolation: isolation, ReadOnly: true})
}

// SQLite TEXT substr treats NUL as an end marker. Read bounded raw bytes and
// preserve the original storage class so neither NUL tails nor BLOBs alias text.
func boundedInspectionText(column string, limit int, driver string) string {
	size := strconv.Itoa(limit)
	if driver == "sqlite" {
		return "CASE WHEN typeof(" + column + ")='text' THEN substr(CAST(" + column + " AS BLOB),1," + size + ") END"
	}
	return "substr(" + column + ",1," + size + ")"
}
