package access

import (
	"context"
	"errors"

	"github.com/elcomware/edupilot/modules/core/scope"
)

const HierarchyRead = "tenancy.hierarchy.read"

var ErrInvalidScope = errors.New("invalid scoped request")

// Subject preserves realm-local identity, including historical non-UUID realms.
type Subject struct {
	Realm string `json:"realm"`
	ID    string `json:"id"`
}

// TenantAuthorization is issued from Auth's current control snapshot. It is
// request-local and must never be persisted or accepted from transport input.
type TenantAuthorization struct {
	Subject    Subject        `json:"subject"`
	Tenant     scope.TenantID `json:"tenant"`
	Permission string         `json:"permission"`
	GrantID    string         `json:"grantId"`
	Version    uint64         `json:"grantVersion"`
	ExpiresAt  int64          `json:"expiresAt"`
}

type TenantAuthorizer interface {
	AuthorizeTenant(context.Context, string, []string, scope.TenantID) (TenantAuthorization, error)
}
