package identity

import (
	"context"
	"errors"
	"github.com/elcomware/edupilot/platform/access"
)

const (
	DocumentsRead  = access.DocumentsRead
	DocumentsWrite = access.DocumentsWrite
	StudentsRead   = access.StudentsRead
	StudentsWrite  = access.StudentsWrite
	WorkspaceRead  = access.WorkspaceRead
	WorkspaceWrite = access.WorkspaceWrite
	ModulesRead    = access.ModulesRead
	ManageMembers  = access.ManageMembers
	RoleAdmin      = "admin"
	RoleRegistrar  = "registrar"
	RoleViewer     = "viewer"
)

var (
	ErrInvalidCredentials = errors.New("invalid school, email or password")
	ErrUnauthenticated    = access.ErrUnauthenticated
	ErrForbidden          = access.ErrForbidden
	ErrInvalidInput       = errors.New("invalid school, account or password")
	ErrConflict           = errors.New("account or school already provisioned")
	ErrNotFound           = errors.New("school member not found")
)

type Principal = access.Principal

func principal(id, school, role string) (Principal, error) {
	var grants []string
	switch role {
	case RoleAdmin:
		grants = []string{ModulesRead, StudentsRead, StudentsWrite, WorkspaceRead, WorkspaceWrite, DocumentsRead, DocumentsWrite, ManageMembers}
	case RoleRegistrar:
		grants = []string{ModulesRead, StudentsRead, StudentsWrite, WorkspaceRead, WorkspaceWrite, DocumentsRead, DocumentsWrite}
	case RoleViewer:
		grants = []string{ModulesRead, StudentsRead, WorkspaceRead, DocumentsRead}
	default:
		return Principal{}, ErrForbidden
	}
	return Principal{ID: id, School: school, Permissions: grants}, nil
}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return access.WithPrincipal(ctx, p)
}
func FromContext(ctx context.Context) (Principal, bool) { return access.FromContext(ctx) }
