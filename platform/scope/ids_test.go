package scope_test

import (
	"encoding/json"
	"github.com/elcomware/edupilot-os/platform/scope"
	"testing"
)

func TestScopeIDParsing(t *testing.T) {
	const valid = "1c89e523-51f3-4c88-8e4c-62591a336ada"
	for _, v := range []string{"", "00000000-0000-0000-0000-000000000000", "legacy-school-7", "1C89E523-51F3-4C88-8E4C-62591A336ADA", "{" + valid + "}", "1c89e523-51f3-4c88-0e4c-62591a336ada"} {
		if _, err := scope.Parse[scope.Tenant](v); err == nil {
			t.Errorf("accepted %q", v)
		}
	}
	id, err := scope.Parse[scope.Tenant](valid)
	if err != nil || id.String() != valid {
		t.Fatalf("parse: %v %v", id, err)
	}
	data, err := json.Marshal(id)
	if err != nil || string(data) != "\""+valid+"\"" {
		t.Fatalf("wire encoding: %s %v", data, err)
	}
	var decoded scope.TenantID
	if err = json.Unmarshal(data, &decoded); err != nil || decoded != id {
		t.Fatal("wire roundtrip failed", err)
	}
	if err = json.Unmarshal([]byte(`"bad"`), &decoded); err == nil || decoded != id {
		t.Fatal("invalid decode mutated a valid id")
	}
	a, err := scope.NewID[scope.Tenant]()
	if err != nil || a.IsZero() || a == id {
		t.Fatal("new tenant identity", err)
	}
}
func TestAbsentOwnerIsNotWildcard(t *testing.T) {
	var empty scope.Owner
	if empty.Valid() {
		t.Fatal("zero owner valid")
	}
	tenant, _ := scope.NewID[scope.Tenant]()
	org, _ := scope.NewID[scope.Organization]()
	school, _ := scope.NewID[scope.School]()
	chain := scope.Chain{Tenant: tenant, Organization: org}
	if _, e := chain.Owner(scope.SchoolOwner); e == nil {
		t.Fatal("missing school became tenant-wide")
	}
	chain.School = school
	group, e := chain.Owner(scope.TenantOwner)
	if e != nil {
		t.Fatal(e)
	}
	local, e := chain.Owner(scope.SchoolOwner)
	if e != nil || group == local {
		t.Fatal("incorrect ownership", e)
	}
}
