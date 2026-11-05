// Package sqlstore implements the common work-list contract using each module's own table.
package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/elcomware/edupilot-os/database/edumigrations"
	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/elcomware/edupilot-os/platform/outbox"
	"github.com/elcomware/edupilot-os/platform/workspace"
	"github.com/google/uuid"
)

var safeName = regexp.MustCompile("^[a-z][a-z0-9_]{0,39}$")

func tableName(module string) string {
	// SQL identifiers cannot be bound as parameters. Only composition-time keys pass here.
	if !safeName.MatchString(module) {
		panic("invalid module identifier")
	}
	return "ws_" + module
}
func Schema(module string) edumigrations.Plan {
	table := tableName(module)
	return edumigrations.Plan{ID: "20260102000001_workspace_" + module, Up: []string{
		"CREATE TABLE " + table + " (school_id TEXT NOT NULL, id TEXT NOT NULL, title TEXT NOT NULL, status TEXT NOT NULL CHECK (status IN ('open','closed')), version BIGINT NOT NULL CHECK (version > 0), PRIMARY KEY (school_id,id))",
	}, Down: []string{"DROP TABLE " + table}}
}

type Store struct {
	db            *database.Catalog
	module, table string
}

func New(db *database.Catalog, module string) *Store {
	return &Store{db: db, module: module, table: tableName(module)}
}

func (s *Store) List(ctx context.Context, school string, limit, offset int) (workspace.Page, error) {
	pool, err := s.db.ForTenantContext(ctx, school)
	if err != nil {
		return workspace.Page{}, err
	}
	// A single statement gives the total and page the same snapshot, even at the end of the list.
	q := fmt.Sprintf("WITH page AS (SELECT id,title,status,version FROM %s WHERE school_id=? ORDER BY id LIMIT ? OFFSET ?) SELECT totals.n,page.id,page.title,page.status,page.version FROM (SELECT COUNT(*) AS n FROM %s WHERE school_id=?) totals LEFT JOIN page ON 1=1 ORDER BY page.id", s.table, s.table)
	rows, err := pool.DB.QueryContext(ctx, pool.Bind(q), school, limit, offset, school)
	if err != nil {
		return workspace.Page{}, err
	}
	defer rows.Close()
	page := workspace.Page{Items: []workspace.Entry{}}
	for rows.Next() {
		var id, title, status sql.NullString
		var version sql.NullInt64
		if err := rows.Scan(&page.Total, &id, &title, &status, &version); err != nil {
			return workspace.Page{}, err
		}
		if id.Valid {
			page.Items = append(page.Items, workspace.Entry{ID: id.String, Title: title.String, Status: status.String, Version: version.Int64})
		}
	}
	return page, rows.Err()
}
func (s *Store) Create(ctx context.Context, school, title string) (workspace.Entry, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return workspace.Entry{}, err
	}
	item := workspace.Entry{ID: id.String(), Title: title, Status: "open", Version: 1}
	err = s.mutate(ctx, school, item, "created", func(tx *sql.Tx, pool database.Pool) error {
		_, err := tx.ExecContext(ctx, pool.Bind("INSERT INTO "+s.table+" (school_id,id,title,status,version) VALUES (?,?,?,?,?)"), school, item.ID, item.Title, item.Status, item.Version)
		return err
	})
	return item, err
}
func (s *Store) Update(ctx context.Context, school string, item workspace.Entry) (workspace.Entry, error) {
	expected := item.Version
	item.Version++
	err := s.mutate(ctx, school, item, "updated", func(tx *sql.Tx, pool database.Pool) error {
		result, err := tx.ExecContext(ctx, pool.Bind("UPDATE "+s.table+" SET title=?,status=?,version=? WHERE school_id=? AND id=? AND version=?"), item.Title, item.Status, item.Version, school, item.ID, expected)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil || n == 1 {
			return err
		}
		var current int64
		err = tx.QueryRowContext(ctx, pool.Bind("SELECT version FROM "+s.table+" WHERE school_id=? AND id=?"), school, item.ID).Scan(&current)
		if errors.Is(err, sql.ErrNoRows) {
			return workspace.ErrMissing
		}
		if err != nil {
			return err
		}
		return workspace.ErrConflict
	})
	return item, err
}
func (s *Store) mutate(ctx context.Context, school string, item workspace.Entry, action string, write func(*sql.Tx, database.Pool) error) error {
	pool, err := s.db.ForTenantContext(ctx, school)
	if err != nil {
		return err
	}
	tx, err := pool.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = write(tx, pool); err != nil {
		return err
	}
	payload, err := json.Marshal(struct {
		Version int64 `json:"version"`
	}{item.Version})
	if err != nil {
		return err
	}
	if err = outbox.Append(ctx, tx, pool, outbox.Event{TenantID: school, AggregateID: item.ID, Type: s.module + ".workspace." + action + ".v1", Payload: payload}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Get(ctx context.Context, school, id string) (workspace.Entry, error) {
	pool, err := s.db.ForTenantContext(ctx, school)
	if err != nil {
		return workspace.Entry{}, err
	}
	var item workspace.Entry
	err = pool.DB.QueryRowContext(ctx, pool.Bind("SELECT id,title,status,version FROM "+s.table+" WHERE school_id=? AND id=?"), school, id).Scan(&item.ID, &item.Title, &item.Status, &item.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return workspace.Entry{}, workspace.ErrMissing
	}
	return item, err
}
