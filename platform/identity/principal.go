package identity

import (
	"context"
	"errors"
	"slices"
)

const (
	StudentsRead   = "students.read"
	StudentsWrite  = "students.write"
	WorkspaceRead  = "workspace.read"
	WorkspaceWrite = "workspace.write"
	ModulesRead    = "modules.read"
	ManageMembers  = "identity.members.manage"

	RoleAdmin     = "admin"
	RoleRegistrar = "registrar"
	RoleViewer    = "viewer"
)

var (
	ErrInvalidCredentials = errors.New("invalid school, email or password")
	ErrUnauthenticated    = errors.New("invalid or expired session")
	ErrForbidden          = errors.New("access denied")
	ErrInvalidInput       = errors.New("invalid school, account or password")
	ErrConflict           = errors.New("account or school already provisioned")
	ErrNotFound           = errors.New("school member not found")
)

type Principal struct {
	ID          string   `json:"id"`
	School      string   `json:"school"`
	Permissions []string `json:"permissions"`
}

func (p Principal) Allows(permission string) bool {
	return p.ID != "" && p.School != "" && slices.Contains(p.Permissions, permission)
}

// Require checks the requested tenant and action together, including for admins.
func (p Principal) Require(school, permission string) error {
	if school != p.School || !p.Allows(permission) {
		return ErrForbidden
	}
	return nil
}

func principal(id, school, role string) (Principal, error) {
	var grants []string
	switch role {
	case RoleAdmin:
		grants = []string{ModulesRead, StudentsRead, StudentsWrite, WorkspaceRead, WorkspaceWrite, ManageMembers}
	case RoleRegistrar:
		grants = []string{ModulesRead, StudentsRead, StudentsWrite, WorkspaceRead, WorkspaceWrite}
	case RoleViewer:
		grants = []string{ModulesRead, StudentsRead, WorkspaceRead}
	default:
		return Principal{}, ErrForbidden
	}
	return Principal{ID: id, School: school, Permissions: grants}, nil
}

type contextKey struct{}

// WithPrincipal is for trusted authentication adapters and in-process use cases.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, p)
}

func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(contextKey{}).(Principal)
	return p, ok && p.ID != "" && p.School != ""
}
