package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/modules/enrollment/internal/domain"
)

func TestPlacementValidationAndConflictingRetries(t *testing.T) {
	for _, input := range []struct{ year, grade string }{
		{"2026/27", "Grade 8"}, {"2026-2028", "Grade 8"}, {"2026-2027", " "}, {"2026-2027", strings.Repeat("界", 49)},
	} {
		if _, err := domain.NewPlacement("id", "student", input.year, input.grade, ""); !errors.Is(err, domain.ErrInvalid) {
			t.Fatalf("%+v: %v", input, err)
		}
	}
	p, err := domain.NewPlacement("id", "student", "2026-2027", "  Grade 8 ", "General")
	if err != nil {
		t.Fatal(err)
	}
	same, err := domain.NewPlacement("another-id", "student", "2026-2027", "Grade 8", "General")
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Retry(same); err != nil {
		t.Fatalf("same natural key and data: %v", err)
	}
	same.Course = "Honours"
	if err = p.Retry(same); !errors.Is(err, domain.ErrYearOccupied) {
		t.Fatalf("course changed: %v", err)
	}
	p, err = p.Withdraw(1)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Retry(same); !errors.Is(err, domain.ErrWithdrawn) {
		t.Fatalf("retry after withdrawal: %v", err)
	}
	if _, err = p.Withdraw(1); !errors.Is(err, domain.ErrStale) {
		t.Fatalf("stale withdrawal: %v", err)
	}
}
