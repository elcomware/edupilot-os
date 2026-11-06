package models_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/modules/enrollment/app/models"
)

func TestPlacementValidationAndConflictingRetries(t *testing.T) {
	for _, input := range []struct{ year, grade string }{
		{"2026/27", "Grade 8"}, {"2026-2028", "Grade 8"}, {"2026-2027", " "}, {"2026-2027", strings.Repeat("界", 49)},
	} {
		if _, err := models.NewPlacement("id", "student", input.year, input.grade, ""); !errors.Is(err, models.ErrInvalid) {
			t.Fatalf("%+v: %v", input, err)
		}
	}
	p, err := models.NewPlacement("id", "student", "2026-2027", "  Grade 8 ", "General")
	if err != nil {
		t.Fatal(err)
	}
	same, err := models.NewPlacement("another-id", "student", "2026-2027", "Grade 8", "General")
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Retry(same); err != nil {
		t.Fatalf("same natural key and data: %v", err)
	}
	same.Course = "Honours"
	if err = p.Retry(same); !errors.Is(err, models.ErrYearOccupied) {
		t.Fatalf("course changed: %v", err)
	}
	p, err = p.Withdraw(1)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Retry(same); !errors.Is(err, models.ErrWithdrawn) {
		t.Fatalf("retry after withdrawal: %v", err)
	}
	if _, err = p.Withdraw(1); !errors.Is(err, models.ErrStale) {
		t.Fatalf("stale withdrawal: %v", err)
	}
}
