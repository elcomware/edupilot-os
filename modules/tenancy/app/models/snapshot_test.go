package models_test

import (
	"encoding/json"
	"testing"

	s "github.com/elcomware/edupilot/modules/core/scope"
	d "github.com/elcomware/edupilot/modules/tenancy/app/models"
)

func TestSnapshotRoundTripPreservesVersionAndAncestors(t *testing.T) {
	f := setup()
	child := must(s.NewID[s.Organization]())
	g := must(f.g.AddOrganization(must(d.NewOrganization(child, f.t, f.o, "Division"))))
	row := g.Snapshot()
	if row.Version <= f.g.Version() || row.Organizations[0].ID() != f.o {
		t.Fatal("snapshot version or parent order")
	}
	row.Version = 42
	row.Organizations[0] = row.Organizations[0].WithActive(false)
	restored := must(d.Restore(row))
	chain := must(restored.Resolve(s.Selection{Tenant: f.t, Organization: child}))
	if chain.HierarchyVersion != 42 || chain.Active || len(chain.Ancestors) != 2 || chain.Ancestors[0] != f.o {
		t.Fatal(chain)
	}
	original := must(g.Resolve(s.Selection{Tenant: f.t, Organization: child}))
	if !original.Active {
		t.Fatal("edited snapshot mutated live graph")
	}
	row.Organizations = nil
	if _, err := d.Restore(row); err == nil {
		t.Fatal("restored dangling school")
	}
}

func TestSelectionWireAndOwner(t *testing.T) {
	f := setup()
	sel := s.Selection{Tenant: f.t, School: f.a}
	data, err := json.Marshal(sel)
	if err != nil {
		t.Fatal(err)
	}
	var out s.Selection
	if err = json.Unmarshal(data, &out); err != nil || out != sel {
		t.Fatal(out, err)
	}
	if err = json.Unmarshal([]byte(`{"school":null}`), &out); err == nil {
		t.Fatal("null ID accepted")
	}
	chain := must(f.g.Resolve(sel))
	owner, err := chain.Owner(s.SchoolOwner)
	if err != nil || owner.School() != f.a || owner.Organization() != f.o || owner.Tenant() != f.t || !owner.Valid() {
		t.Fatal(owner, err)
	}
	if _, err = chain.Owner(s.CampusOwner); err == nil {
		t.Fatal("missing campus widened ownership")
	}
}

func TestRestoreRejectsVersionBelowPersistedNodeCount(t *testing.T) {
	row := setup().g.Snapshot()
	row.Version--
	if _, err := d.Restore(row); err == nil {
		t.Fatal("accepted a hierarchy version lower than the history required to create its nodes")
	}
}
