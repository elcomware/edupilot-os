package domain

import (
	"maps"
	"math"
	"slices"

	"github.com/elcomware/edupilot/platform/scope"
)

// Graph is an immutable topology snapshot for one tenant. Insert parents
// before children. An existing node cannot be overwritten or reparented.
type Graph struct {
	version       uint64
	active        bool
	tenant        scope.TenantID
	organizations map[scope.OrganizationID]Organization
	schools       map[scope.SchoolID]School
	campuses      map[scope.CampusID]Campus
}

func NewGraph(tenant scope.TenantID) (Graph, error) {
	if tenant.IsZero() {
		return Graph{}, ErrInvalid
	}
	return Graph{version: 1, active: true, tenant: tenant, organizations: make(map[scope.OrganizationID]Organization), schools: make(map[scope.SchoolID]School), campuses: make(map[scope.CampusID]Campus)}, nil
}
func (g Graph) Tenant() scope.TenantID { return g.tenant }
func (g Graph) AddOrganization(o Organization) (Graph, error) {
	if g.version >= math.MaxInt64 {
		return g, ErrConflict
	}
	if g.tenant.IsZero() || o.tenant != g.tenant {
		return g, ErrScope
	}
	if _, exists := g.organizations[o.id]; exists {
		return g, ErrConflict
	}
	if !o.parent.IsZero() {
		if _, ok := g.organizations[o.parent]; !ok {
			return g, ErrScope
		}
	}
	g.organizations = maps.Clone(g.organizations)
	g.organizations[o.id] = o
	g.version++
	return g, nil
}
func (g Graph) AddSchool(s School) (Graph, error) {
	if g.version >= math.MaxInt64 {
		return g, ErrConflict
	}
	if g.tenant.IsZero() || s.tenant != g.tenant {
		return g, ErrScope
	}
	if _, ok := g.organizations[s.organization]; !ok {
		return g, ErrScope
	}
	if _, exists := g.schools[s.id]; exists {
		return g, ErrConflict
	}
	for _, school := range g.schools {
		if school.organization == s.organization && school.code == s.code {
			return g, ErrConflict
		}
	}
	g.schools = maps.Clone(g.schools)
	g.schools[s.id] = s
	g.version++
	return g, nil
}
func (g Graph) AddCampus(c Campus) (Graph, error) {
	if g.version >= math.MaxInt64 {
		return g, ErrConflict
	}
	if g.tenant.IsZero() || c.tenant != g.tenant {
		return g, ErrScope
	}
	if _, ok := g.schools[c.school]; !ok {
		return g, ErrScope
	}
	if _, exists := g.campuses[c.id]; exists {
		return g, ErrConflict
	}
	for _, campus := range g.campuses {
		if campus.school == c.school && campus.code == c.code {
			return g, ErrConflict
		}
	}
	g.campuses = maps.Clone(g.campuses)
	g.campuses[c.id] = c
	g.version++
	return g, nil
}
func (g Graph) Resolve(sel scope.Selection) (scope.Chain, error) {
	if g.tenant.IsZero() || sel.Tenant != g.tenant {
		return scope.Chain{}, ErrScope
	}
	if !sel.Team.IsZero() {
		return scope.Chain{}, ErrTeam
	}
	active := g.active
	if !sel.Campus.IsZero() {
		campus, ok := g.campuses[sel.Campus]
		if !ok || (!sel.School.IsZero() && campus.school != sel.School) {
			return scope.Chain{}, ErrScope
		}
		active = active && campus.active
		sel.School = campus.school
	}
	if !sel.School.IsZero() {
		school, ok := g.schools[sel.School]
		if !ok || (!sel.Organization.IsZero() && school.organization != sel.Organization) {
			return scope.Chain{}, ErrScope
		}
		active = active && school.active
		sel.Organization = school.organization
	}
	chain := scope.Chain{Active: active, HierarchyVersion: g.version, Tenant: g.tenant, Organization: sel.Organization, School: sel.School, Campus: sel.Campus}
	for id := sel.Organization; !id.IsZero(); {
		org, ok := g.organizations[id]
		if !ok {
			return scope.Chain{}, ErrScope
		}
		chain.Active = chain.Active && org.active
		chain.Ancestors = append(chain.Ancestors, id)
		id = org.parent
	}
	slices.Reverse(chain.Ancestors)
	return chain, nil
}
