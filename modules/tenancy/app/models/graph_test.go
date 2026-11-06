package models_test

import (
	"testing"

	s "github.com/elcomware/edupilot/modules/core/scope"
	d "github.com/elcomware/edupilot/modules/tenancy/app/models"
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

type fixture struct {
	g    d.Graph
	t    s.TenantID
	o    s.OrganizationID
	a, b s.SchoolID
}

func setup() fixture {
	f := fixture{t: must(s.NewID[s.Tenant]()), o: must(s.NewID[s.Organization]()), a: must(s.NewID[s.School]()), b: must(s.NewID[s.School]())}
	f.g = must(d.NewGraph(f.t))
	f.g = must(f.g.AddOrganization(must(d.NewOrganization(f.o, f.t, s.OrganizationID{}, "Écoles du Nord"))))
	for id, code := range map[s.SchoolID]string{f.a: "NORD", f.b: "EST"} {
		f.g = must(f.g.AddSchool(must(d.NewSchool(id, f.t, f.o, code, "École"))))
	}
	return f
}
func TestTwoSchoolsOneTenant(t *testing.T) {
	f := setup()
	for _, id := range []s.SchoolID{f.a, f.b} {
		c, e := f.g.Resolve(s.Selection{Tenant: f.t, School: id})
		if e != nil || c.Tenant != f.t || c.Organization != f.o || c.School != id {
			t.Fatal(c, e)
		}
	}
}
func TestGraphRejectsCrossTenantAncestry(t *testing.T) {
	f := setup()
	other := must(s.NewID[s.Tenant]())
	wrong := must(d.NewSchool(must(s.NewID[s.School]()), other, f.o, "NEW", "New"))
	if _, e := f.g.AddSchool(wrong); e == nil {
		t.Fatal("foreign school accepted")
	}
	unknown := must(s.NewID[s.Organization]())
	if _, e := f.g.AddOrganization(must(d.NewOrganization(unknown, f.t, must(s.NewID[s.Organization]()), "Unknown parent"))); e == nil {
		t.Fatal("dangling parent")
	}
	if _, e := f.g.Resolve(s.Selection{Tenant: other, School: f.a}); e == nil {
		t.Fatal("foreign selection")
	}
	if _, e := f.g.Resolve(s.Selection{Tenant: f.t, Organization: unknown, School: f.a}); e == nil {
		t.Fatal("contradictory org")
	}
	if _, e := f.g.Resolve(s.Selection{}); e == nil {
		t.Fatal("zero tenant")
	}
}
func TestGraphRejectsCycle(t *testing.T) {
	f := setup()
	child := must(s.NewID[s.Organization]())
	g := must(f.g.AddOrganization(must(d.NewOrganization(child, f.t, f.o, "Child"))))
	if _, e := g.AddOrganization(must(d.NewOrganization(f.o, f.t, child, "Cycle"))); e == nil {
		t.Fatal("cycle accepted")
	}
	if _, e := f.g.Resolve(s.Selection{Tenant: f.t, Organization: child}); e == nil {
		t.Fatal("mutation changed earlier graph")
	}
}
func TestCampusCodeUniqueWithinSchool(t *testing.T) {
	f := setup()
	newCampus := func(school s.SchoolID, code string) d.Campus {
		return must(d.NewCampus(must(s.NewID[s.Campus]()), f.t, school, code, "Bâtiment A", "Europe/Paris"))
	}
	a := newCampus(f.a, "MAIN")
	g := must(f.g.AddCampus(a))
	if _, e := g.AddCampus(newCampus(f.a, "main")); e == nil {
		t.Fatal("duplicate campus code")
	}
	if _, e := g.AddCampus(newCampus(f.b, "MAIN")); e != nil {
		t.Fatal(e)
	}
	if _, e := g.Resolve(s.Selection{Tenant: f.t, School: f.b, Campus: a.ID()}); e == nil {
		t.Fatal("campus from sibling school")
	}
	c, e := g.Resolve(s.Selection{Tenant: f.t, Campus: a.ID()})
	if e != nil || c.School != f.a || c.Organization != f.o {
		t.Fatal(c, e)
	}
	if _, e := d.NewCampus(must(s.NewID[s.Campus]()), f.t, f.a, "X", "Campus", "Local"); e == nil {
		t.Fatal("host-dependent timezone")
	}
}
func TestTeamSelectionDoesNotGrantAccess(t *testing.T) {
	f := setup()
	if _, e := f.g.Resolve(s.Selection{Tenant: f.t, School: f.a, Team: must(s.NewID[s.Team]())}); e == nil {
		t.Fatal("topology accepted unverified team narrowing")
	}
}
