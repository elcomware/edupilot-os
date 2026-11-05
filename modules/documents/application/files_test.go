package application

import (
	"context"
	"errors"
	"testing"

	"github.com/elcomware/edupilot-os/platform/identity"
)

func TestUseCaseRequiresVerifiedFilePermission(t *testing.T) {
	files := Files{Resolve: func() (BlobStore, error) { t.Fatal("unauthorized request reached storage"); return nil, nil }}
	if _, err := files.Upload(context.Background(), "text/plain", []byte("notice")); !errors.Is(err, identity.ErrUnauthenticated) {
		t.Fatalf("missing principal: %v", err)
	}
	ctx := identity.WithPrincipal(t.Context(), identity.Principal{ID: "user", School: "north", Permissions: []string{identity.WorkspaceWrite}})
	if _, err := files.Upload(ctx, "text/plain", []byte("notice")); !errors.Is(err, identity.ErrForbidden) {
		t.Fatalf("workspace write granted file upload: %v", err)
	}
	if _, err := files.Download(ctx, "missing"); !errors.Is(err, identity.ErrForbidden) {
		t.Fatalf("workspace write granted file download: %v", err)
	}
}
