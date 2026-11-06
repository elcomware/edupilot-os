package application

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"github.com/elcomware/edupilot/modules/enrollment/internal/domain"
	"github.com/elcomware/edupilot/modules/students/contracts"
)

// A new placement and its active-student check share a transaction. Identical
// retries return the original placement without checking current student status.
type PlacementStore interface {
	Record(ctx context.Context, school, actor string, proposed domain.Placement, check func(context.Context) error) (domain.Placement, bool, error)
	Withdraw(ctx context.Context, school, actor, id string, version int64) (domain.Placement, error)
	List(ctx context.Context, school, student string, limit, offset int) ([]domain.Placement, int64, error)
}

type Placements struct {
	Store    PlacementStore
	Students contracts.Directory
}

func (s Placements) Record(ctx context.Context, school, actor, student, year, grade, course string) (domain.Placement, bool, error) {
	if school == "" || actor == "" {
		return domain.Placement{}, false, fmt.Errorf("%w: school and actor are required", domain.ErrInvalid)
	}
	p, err := domain.NewPlacement(rand.Text(), student, year, grade, course)
	if err != nil {
		return domain.Placement{}, false, err
	}
	check := func(ctx context.Context) error {
		student, err := s.Students.Lookup(ctx, school, p.StudentID)
		if errors.Is(err, contracts.ErrNotFound) {
			return domain.ErrStudentMiss
		}
		if err != nil {
			return err
		}
		if student.Status != "active" {
			return domain.ErrStudentClosed
		}
		return nil
	}
	return s.Store.Record(ctx, school, actor, p, check)
}
func (s Placements) Withdraw(ctx context.Context, school, actor, id string, version int64) (domain.Placement, error) {
	if school == "" || actor == "" || version < 1 {
		return domain.Placement{}, fmt.Errorf("%w: school, actor and positive version are required", domain.ErrInvalid)
	}
	return s.Store.Withdraw(ctx, school, actor, id, version)
}
func (s Placements) List(ctx context.Context, school, student string, limit, offset int) ([]domain.Placement, int64, error) {
	if school == "" || limit < 1 || limit > 100 || offset < 0 || len(student) > 64 || strings.TrimSpace(student) != student {
		return nil, 0, fmt.Errorf("%w: check school, studentId, limit (1 to 100), and offset (nonnegative)", domain.ErrInvalid)
	}
	return s.Store.List(ctx, school, student, limit, offset)
}
