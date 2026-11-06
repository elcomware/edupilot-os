package models

import (
	"strings"

	"github.com/elcomware/edupilot/modules/core/scope"
)

type School struct {
	active       bool
	id           scope.SchoolID
	tenant       scope.TenantID
	organization scope.OrganizationID
	code, name   string
}

func NewSchool(id scope.SchoolID, tenant scope.TenantID, organization scope.OrganizationID, code, name string) (School, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	name = strings.TrimSpace(name)
	if id.IsZero() || tenant.IsZero() || organization.IsZero() || !codeFormat.MatchString(code) || !validName(name) {
		return School{}, ErrInvalid
	}
	return School{true, id, tenant, organization, code, name}, nil
}
func (s School) ID() scope.SchoolID                 { return s.id }
func (s School) Tenant() scope.TenantID             { return s.tenant }
func (s School) Organization() scope.OrganizationID { return s.organization }
func (s School) Code() string                       { return s.code }
func (s School) Name() string                       { return s.name }

func (s School) Active() bool                  { return s.active }
func (s School) WithActive(active bool) School { s.active = active; return s }
