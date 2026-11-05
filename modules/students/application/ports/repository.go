package ports

import (
	"context"

	"github.com/elcomware/edupilot-os/modules/students/domain"
)

// Repository is the application transaction boundary. Insert and Replace must
// commit the student and its event together. Replace compares the persisted
// version to expected in the database, even after the use case validated it.
type Repository interface {
	Insert(ctx context.Context, s domain.Student, actor string) error
	Get(ctx context.Context, school, id string) (domain.Student, error)
	ActivePage(ctx context.Context, school string, limit, offset int) ([]domain.Student, int64, error)
	Replace(ctx context.Context, s domain.Student, expected int64, actor string) error
}
