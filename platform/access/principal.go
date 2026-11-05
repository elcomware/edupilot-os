// Package access carries a verified school principal without importing authentication or SQL adapters.
package access

import (
	"context"
	"errors"
	"slices"
)

const (
	DocumentsRead  = "documents.read"
	DocumentsWrite = "documents.write"
	StudentsRead   = "students.read"
	StudentsWrite  = "students.write"
	WorkspaceRead  = "workspace.read"
	WorkspaceWrite = "workspace.write"
	ModulesRead    = "modules.read"
	ManageMembers  = "identity.members.manage"
)

var ErrForbidden = errors.New("access denied")
var ErrUnauthenticated = errors.New("invalid or expired session")

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

type contextKey struct{}

// WithPrincipal is for trusted authentication adapters and in-process use cases.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, p)
}

func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(contextKey{}).(Principal)
	return p, ok && p.ID != "" && p.School != ""
}
