package models_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/modules/students/app/models"
)

func TestPersonalNamesPreserveSpellingButRejectInvalidInput(t *testing.T) {
	s, err := models.Register("id", "north", "  Île  ", "O’Neil-Soto")
	if err != nil {
		t.Fatal(err)
	}
	if s.GivenName() != "Île" || s.FamilyName() != "O’Neil-Soto" {
		t.Fatal("spelling changed")
	}
	for _, name := range []string{"\tAda", " \u2002 ", strings.Repeat("界", 101), "Ali\nce"} {
		if _, err := models.Register("id", "north", name, "Family"); !errors.Is(err, models.ErrInvalid) {
			t.Errorf("expected invalid input, got %v", err)
		}
	}
	if _, err := models.Register("id", "north", strings.Repeat("界", 100), "Family"); err != nil {
		t.Fatal(err)
	}
}

func TestArchivedRecordCannotBeReopenedOrChanged(t *testing.T) {
	s, err := models.Register("id", "north", "Ana", "Liu")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Archive(2); !errors.Is(err, models.ErrConflict) {
		t.Fatalf("stale: %v", err)
	}
	archived, err := s.Archive(1)
	if err != nil {
		t.Fatal(err)
	}
	if archived.Version() != 2 || archived.Status() != models.Archived || s.Version() != 1 {
		t.Fatal("bad archive transition")
	}
	if _, err = archived.Rename(2, "Ana", "Chan"); !errors.Is(err, models.ErrArchived) {
		t.Fatalf("rename: %v", err)
	}
	if _, err = archived.Archive(2); !errors.Is(err, models.ErrArchived) {
		t.Fatalf("repeat archive: %v", err)
	}
}
