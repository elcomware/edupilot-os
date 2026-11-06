// Package persistence uses the SQL handle owned by the school's Goravel
// connection. It never opens an independent production pool.
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/elcomware/edupilot/modules/students/application/ports"
	"github.com/elcomware/edupilot/modules/students/contracts"
	"github.com/elcomware/edupilot/modules/students/domain"
	"github.com/elcomware/edupilot/platform/database"
	"github.com/elcomware/edupilot/platform/outbox"
)

// Catalog is implemented by the hybrid tenant connection catalog. Production
// wiring supplies that catalog; tests can also bind a school to an isolated DB.
type Catalog interface {
	ForTenantContext(ctx context.Context, id string) (database.Pool, error)
}

type Repository struct{ catalog Catalog }

func New(catalog Catalog) *Repository { return &Repository{catalog: catalog} }

var _ ports.Repository = (*Repository)(nil)

func (r *Repository) Get(ctx context.Context, school, id string) (domain.Student, error) {
	const query = "SELECT given_name, family_name, status, version FROM students_records WHERE school_id=? AND id=?"
	var row *sql.Row
	if scope, ok := database.TransactionFromContext(ctx); ok {
		if scope.School != school || scope.Tx == nil || scope.Pool.DB == nil {
			return domain.Student{}, errors.New("invalid student lookup transaction scope")
		}
		statement := query
		if scope.Pool.Driver == "postgres" {
			// An enrollment transaction holds this lock until its own write commits.
			statement += " FOR SHARE"
		}
		row = scope.Tx.QueryRowContext(ctx, scope.Pool.Bind(statement), school, id)
	} else {
		pool, err := r.catalog.ForTenantContext(ctx, school)
		if err != nil {
			return domain.Student{}, err
		}
		row = pool.DB.QueryRowContext(ctx, pool.Bind(query), school, id)
	}
	var given, family, status string
	var version int64
	if err := row.Scan(&given, &family, &status, &version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Student{}, domain.ErrNotFound
		}
		return domain.Student{}, fmt.Errorf("get student: %w", err)
	}
	return persisted(id, school, given, family, version, status)
}

func persisted(id, school, given, family string, version int64, status string) (domain.Student, error) {
	s, err := domain.Restore(id, school, given, family, version, domain.Status(status))
	if err != nil {
		// Invalid stored data is an internal error, not client-side validation.
		return domain.Student{}, errors.New("stored student violates domain constraints")
	}
	return s, nil
}

func (r *Repository) ActivePage(ctx context.Context, school string, limit, offset int) ([]domain.Student, int64, error) {
	p, err := r.catalog.ForTenantContext(ctx, school)
	if err != nil {
		return nil, 0, err
	}
	// One snapshot supplies both the total and the rows. The totals row survives
	// even when offset is past the final student.
	rows, err := p.DB.QueryContext(ctx, p.Bind(`
		WITH page AS (
			SELECT id, given_name, family_name, status, version FROM students_records
			WHERE school_id=? AND status='active' ORDER BY family_name, given_name, id LIMIT ? OFFSET ?
		)
		SELECT totals.n, page.id, page.given_name, page.family_name, page.status, page.version
		FROM (SELECT COUNT(*) AS n FROM students_records WHERE school_id=? AND status='active') AS totals
		LEFT JOIN page ON 1=1 ORDER BY page.family_name, page.given_name, page.id`), school, limit, offset, school)
	if err != nil {
		return nil, 0, fmt.Errorf("list students: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Student, 0)
	var total int64
	for rows.Next() {
		var id, given, family, status sql.NullString
		var version sql.NullInt64
		if err := rows.Scan(&total, &id, &given, &family, &status, &version); err != nil {
			return nil, 0, err
		}
		if !id.Valid {
			continue
		}
		s, err := persisted(id.String, school, given.String, family.String, version.Int64, status.String)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *Repository) Insert(ctx context.Context, s domain.Student, actor string) error {
	return r.transaction(ctx, s, contracts.RegisteredV1, actor, func(tx *sql.Tx, p database.Pool) error {
		_, err := tx.ExecContext(ctx, p.Bind(`INSERT INTO students_records (school_id,id,given_name,family_name,status,version) VALUES (?,?,?,?,?,?)`),
			s.School(), s.ID(), s.GivenName(), s.FamilyName(), s.Status(), s.Version())
		return err
	})
}

func (r *Repository) Replace(ctx context.Context, s domain.Student, expected int64, actor string) error {
	event := contracts.UpdatedV1
	if s.Status() == domain.Archived {
		event = contracts.ArchivedV1
	}
	return r.transaction(ctx, s, event, actor, func(tx *sql.Tx, p database.Pool) error {
		res, err := tx.ExecContext(ctx, p.Bind(`UPDATE students_records SET given_name=?,family_name=?,status=?,version=? WHERE school_id=? AND id=? AND version=? AND status='active'`), s.GivenName(), s.FamilyName(), s.Status(), s.Version(), s.School(), s.ID(), expected)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return domain.ErrConflict
		}
		return nil
	})
}

func (r *Repository) transaction(ctx context.Context, s domain.Student, eventType, actor string, write func(*sql.Tx, database.Pool) error) error {
	p, err := r.catalog.ForTenantContext(ctx, s.School())
	if err != nil {
		return err
	}
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = write(tx, p); err != nil {
		return fmt.Errorf("write student: %w", err)
	}
	// Avoid disclosing a student's name through the queue; consumers can use
	// the school-scoped Directory if they need an authorized current view.
	payload, err := json.Marshal(contracts.LifecycleEventV1{
		StudentID: s.ID(), Version: s.Version(), Status: string(s.Status()), ActorID: actor,
	})
	if err != nil {
		return err
	}
	if err = outbox.Append(ctx, tx, p, outbox.Event{TenantID: s.School(), Type: eventType, AggregateID: s.ID(), Payload: payload}); err != nil {
		return fmt.Errorf("append student event: %w", err)
	}
	return tx.Commit()
}
