package models

import (
	"strings"
	"time"
	_ "time/tzdata" // Keep IANA zone validation available in minimal distributions.

	"github.com/elcomware/edupilot/platform/scope"
)

type Campus struct {
	active               bool
	id                   scope.CampusID
	tenant               scope.TenantID
	school               scope.SchoolID
	code, name, timezone string
}

func NewCampus(id scope.CampusID, tenant scope.TenantID, school scope.SchoolID, code, name, timezone string) (Campus, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	name = strings.TrimSpace(name)
	if id.IsZero() || tenant.IsZero() || school.IsZero() || !codeFormat.MatchString(code) || !validName(name) || timezone == "" || timezone == "Local" {
		return Campus{}, ErrInvalid
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return Campus{}, ErrInvalid
	}
	return Campus{true, id, tenant, school, code, name, timezone}, nil
}
func (c Campus) ID() scope.CampusID     { return c.id }
func (c Campus) Tenant() scope.TenantID { return c.tenant }
func (c Campus) School() scope.SchoolID { return c.school }
func (c Campus) Code() string           { return c.code }
func (c Campus) Name() string           { return c.name }
func (c Campus) Timezone() string       { return c.timezone }

func (c Campus) Active() bool                  { return c.active }
func (c Campus) WithActive(active bool) Campus { c.active = active; return c }
