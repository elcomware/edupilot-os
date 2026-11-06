package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/elcomware/edupilot/modules/enrollment/application"
	"github.com/elcomware/edupilot/modules/enrollment/internal/domain"
	"github.com/elcomware/edupilot/platform/database"
	"github.com/elcomware/edupilot/platform/outbox"
)

type PlacementCatalog interface {
	ForTenantContext(context.Context, string) (database.Pool, error)
}

type PlacementRepository struct{ Catalog PlacementCatalog }

var _ application.PlacementStore = PlacementRepository{}

const placementColumns = "id,student_id,academic_year,grade,course,status,version"

func scanPlacement(row *sql.Row) (domain.Placement, error) {
	var p domain.Placement
	err := row.Scan(&p.ID, &p.StudentID, &p.AcademicYear, &p.Grade, &p.Course, &p.Status, &p.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return p, domain.ErrPlacementMiss
	}
	return p, err
}

// Record reserves the unique key before checking Students. Concurrent retries
// contend on that key and can never publish a second registration event.
func (r PlacementRepository) Record(ctx context.Context, school, actor string, p domain.Placement, check func(context.Context) error) (domain.Placement, bool, error) {
	pool, err := r.Catalog.ForTenantContext(ctx, school)
	if err != nil {
		return domain.Placement{}, false, err
	}
	tx, err := pool.DB.BeginTx(ctx, nil)
	if err != nil {
		return domain.Placement{}, false, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, pool.Bind(`INSERT INTO enrollment_placements
		(school_id,id,student_id,academic_year,grade,course,status,version)
		VALUES(?,?,?,?,?,?,?,?) ON CONFLICT (school_id,student_id,academic_year) DO NOTHING`), school, p.ID, p.StudentID, p.AcademicYear, p.Grade, p.Course, p.Status, p.Version)
	if err != nil {
		return domain.Placement{}, false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return domain.Placement{}, false, err
	}
	if n == 0 {
		prior, err := scanPlacement(tx.QueryRowContext(ctx, pool.Bind("SELECT "+placementColumns+" FROM enrollment_placements WHERE school_id=? AND student_id=? AND academic_year=?"), school, p.StudentID, p.AcademicYear))
		if err != nil {
			return domain.Placement{}, false, err
		}
		if err = prior.Retry(p); err != nil {
			return domain.Placement{}, false, err
		}
		return prior, false, tx.Commit()
	}
	scope := database.TenantTransaction{School: school, Pool: pool, Tx: tx}
	if err = check(database.WithTenantTransaction(ctx, scope)); err != nil {
		return domain.Placement{}, false, err
	}
	if err = placementEvent(ctx, tx, pool, school, actor, "enrollment.placed.v1", p); err != nil {
		return domain.Placement{}, false, err
	}
	return p, true, tx.Commit()
}

func (r PlacementRepository) Withdraw(ctx context.Context, school, actor, id string, version int64) (domain.Placement, error) {
	pool, err := r.Catalog.ForTenantContext(ctx, school)
	if err != nil {
		return domain.Placement{}, err
	}
	tx, err := pool.DB.BeginTx(ctx, nil)
	if err != nil {
		return domain.Placement{}, err
	}
	defer tx.Rollback()
	// Start with an atomic write (also acquires SQLite's write lock before any
	// reads). Only a failed match needs the current state to classify the error.
	res, err := tx.ExecContext(ctx, pool.Bind(`UPDATE enrollment_placements SET status='withdrawn', version=version+1
		WHERE school_id=? AND id=? AND version=? AND status='active'`), school, id, version)
	if err != nil {
		return domain.Placement{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return domain.Placement{}, err
	}
	p, err := scanPlacement(tx.QueryRowContext(ctx, pool.Bind("SELECT "+placementColumns+" FROM enrollment_placements WHERE school_id=? AND id=?"), school, id))
	if err != nil {
		return domain.Placement{}, err
	}
	if n == 0 {
		_, err = p.Withdraw(version)
		if err == nil {
			err = domain.ErrStale
		}
		return domain.Placement{}, err
	}
	if err = placementEvent(ctx, tx, pool, school, actor, "enrollment.withdrawn.v1", p); err != nil {
		return domain.Placement{}, err
	}
	return p, tx.Commit()
}

func placementEvent(ctx context.Context, tx *sql.Tx, pool database.Pool, school, actor, kind string, p domain.Placement) error {
	payload, err := json.Marshal(struct {
		PlacementID  string `json:"placementId"`
		StudentID    string `json:"studentId"`
		AcademicYear string `json:"academicYear"`
		Version      int64  `json:"version"`
		Status       string `json:"status"`
		ActorID      string `json:"actorId"`
	}{p.ID, p.StudentID, p.AcademicYear, p.Version, p.Status, actor})
	if err != nil {
		return err
	}
	return outbox.Append(ctx, tx, pool, outbox.Event{TenantID: school, AggregateID: p.ID, Type: kind, Payload: payload})
}

func (r PlacementRepository) List(ctx context.Context, school, student string, limit, offset int) ([]domain.Placement, int64, error) {
	pool, err := r.Catalog.ForTenantContext(ctx, school)
	if err != nil {
		return nil, 0, err
	}
	filter := "school_id=?"
	args := []any{school}
	if student != "" {
		filter += " AND student_id=?"
		args = append(args, student)
	}
	args = append(args, limit, offset)
	query := `WITH matching AS (
		SELECT ` + placementColumns + ` FROM enrollment_placements WHERE ` + filter + `
	), page AS (
		SELECT * FROM matching ORDER BY academic_year DESC, id LIMIT ? OFFSET ?
	)
	SELECT totals.n, COALESCE(p.id,''), COALESCE(p.student_id,''), COALESCE(p.academic_year,''),
		COALESCE(p.grade,''), COALESCE(p.course,''), COALESCE(p.status,''), COALESCE(p.version,0)
	FROM (SELECT COUNT(*) AS n FROM matching) AS totals LEFT JOIN page p ON 1=1
	ORDER BY p.academic_year DESC, p.id`
	rows, err := pool.DB.QueryContext(ctx, pool.Bind(query), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []domain.Placement{}
	var total int64
	for rows.Next() {
		var p domain.Placement
		if err = rows.Scan(&total, &p.ID, &p.StudentID, &p.AcademicYear, &p.Grade, &p.Course, &p.Status, &p.Version); err != nil {
			return nil, 0, err
		}
		if p.ID != "" {
			items = append(items, p)
		}
	}
	return items, total, rows.Err()
}
