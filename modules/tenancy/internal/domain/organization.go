package domain

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/elcomware/edupilot-os/platform/scope"
)

var (
	ErrInvalid  = errors.New("tenancy: invalid topology value")
	ErrConflict = errors.New("tenancy: conflicting identity or code")
	ErrScope    = scope.ErrUnknown
	ErrTeam     = scope.ErrUnsupported
	codeFormat  = regexp.MustCompile(`^[A-Z0-9][A-Z0-9._-]{0,63}$`)
)

func validName(name string) bool {
	return name != "" && utf8.ValidString(name) && utf8.RuneCountInString(name) <= 200 && strings.IndexFunc(name, unicode.IsControl) < 0
}

type Organization struct {
	active bool
	id     scope.OrganizationID
	tenant scope.TenantID
	parent scope.OrganizationID
	name   string
}

func NewOrganization(id scope.OrganizationID, tenant scope.TenantID, parent scope.OrganizationID, name string) (Organization, error) {
	name = strings.TrimSpace(name)
	if id.IsZero() || tenant.IsZero() || id == parent || !validName(name) {
		return Organization{}, ErrInvalid
	}
	return Organization{true, id, tenant, parent, name}, nil
}
func (o Organization) ID() scope.OrganizationID     { return o.id }
func (o Organization) Tenant() scope.TenantID       { return o.tenant }
func (o Organization) Parent() scope.OrganizationID { return o.parent }
func (o Organization) Name() string                 { return o.name }

func (o Organization) Active() bool                        { return o.active }
func (o Organization) WithActive(active bool) Organization { o.active = active; return o }
