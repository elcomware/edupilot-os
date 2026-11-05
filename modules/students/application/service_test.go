package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/elcomware/edupilot-os/modules/students/application"
	"github.com/elcomware/edupilot-os/modules/students/contracts"
	"github.com/elcomware/edupilot-os/modules/students/domain"
	"github.com/elcomware/edupilot-os/modules/students/tests/testkit"
)

func TestRegisterAndMutateKeepScopeVersionAndActor(t *testing.T) {
	ctx := context.Background()
	repo := testkit.NewMemory()
	svc := application.New(repo)
	student, err := svc.Register(ctx, "north", "registrar", "Alba", "Moreau")
	if err != nil {
		t.Fatal(err)
	}
	if student.ID == "" || student.Version != 1 {
		t.Fatalf("bad registration: %+v", student)
	}
	if _, err := svc.Lookup(ctx, "south", student.ID); !errors.Is(err, contracts.ErrNotFound) {
		t.Fatalf("cross-school lookup: %v", err)
	}
	if _, err := svc.Update(ctx, "south", "admin", student.ID, 1, "A.", "Moreau"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-school mutation: %v", err)
	}
	updated, err := svc.Update(ctx, "north", "editor", student.ID, 1, "A.", "Moreau")
	if err != nil || updated.Version != 2 {
		t.Fatalf("update: %+v, %v", updated, err)
	}
	if err := svc.Archive(ctx, "north", "archiver", student.ID, 1); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("stale archive: %v", err)
	}
	if err := svc.Archive(ctx, "north", "archiver", student.ID, 2); err != nil {
		t.Fatal(err)
	}
	if len(repo.Writes) != 3 || repo.Writes[0] != "registrar" || repo.Writes[1] != "editor" || repo.Writes[2] != "archiver" {
		t.Fatalf("unexpected writes: %v", repo.Writes)
	}
	page, err := svc.List(ctx, "north", 25, 0)
	if err != nil || page.Total != 0 || page.Items == nil {
		t.Fatalf("active page: %+v, %v", page, err)
	}
	if _, err := svc.Register(ctx, "north", "registrar", "   ", "Ho"); !errors.Is(err, domain.ErrInvalid) || len(repo.Writes) != 3 {
		t.Fatalf("invalid name wrote a record: %v", err)
	}
}
