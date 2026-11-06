package models

import (
	"cmp"
	"maps"
	"math"
	"slices"

	"github.com/elcomware/edupilot/platform/scope"
)

// Snapshot is a detached persistence boundary. Organizations are emitted
// parent-first. Database adapters can use this order for immediate foreign keys.
type Snapshot struct {
	Tenant        scope.TenantID
	Version       uint64
	Active        bool
	Organizations []Organization
	Schools       []School
	Campuses      []Campus
}

func (g Graph) Version() uint64 { return g.version }

func (g Graph) Snapshot() Snapshot {
	row := Snapshot{Tenant: g.tenant, Version: g.version, Active: g.active}
	// Sort siblings before walking levels, independent of map iteration order.
	children := make(map[scope.OrganizationID][]Organization)
	for _, org := range g.organizations {
		children[org.parent] = append(children[org.parent], org)
	}
	for _, siblings := range children {
		slices.SortFunc(siblings, func(a, b Organization) int { return cmp.Compare(a.id.String(), b.id.String()) })
	}
	row.Organizations = append(row.Organizations, children[scope.OrganizationID{}]...)
	for i := 0; i < len(row.Organizations); i++ {
		row.Organizations = append(row.Organizations, children[row.Organizations[i].id]...)
	}
	row.Schools = slices.SortedFunc(maps.Values(g.schools), func(a, b School) int { return cmp.Compare(a.id.String(), b.id.String()) })
	row.Campuses = slices.SortedFunc(maps.Values(g.campuses), func(a, b Campus) int { return cmp.Compare(a.id.String(), b.id.String()) })
	return row
}

// Restore checks every stored relationship with the same rules used at creation.
// A malformed or unordered snapshot fails before it becomes visible to a caller.
func Restore(row Snapshot) (Graph, error) {
	if row.Version == 0 || row.Version > math.MaxInt64 {
		return Graph{}, ErrInvalid
	}
	graph, err := NewGraph(row.Tenant)
	if err != nil {
		return Graph{}, err
	}
	for _, org := range row.Organizations {
		graph, err = graph.AddOrganization(org)
		if err != nil {
			return Graph{}, err
		}
	}
	for _, school := range row.Schools {
		graph, err = graph.AddSchool(school)
		if err != nil {
			return Graph{}, err
		}
	}
	for _, campus := range row.Campuses {
		graph, err = graph.AddCampus(campus)
		if err != nil {
			return Graph{}, err
		}
	}
	graph.version = row.Version
	graph.active = row.Active
	return graph, nil
}
