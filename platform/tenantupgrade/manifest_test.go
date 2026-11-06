package tenantupgrade

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func mapping(school string, offset byte) Mapping {
	uuid := func(n byte) string { return "f09cdbb2-124b-4f77-87b5-0dc8e053615" + string(n+offset) }
	return Mapping{LegacySchool: school, SourcePool: "shared", SourceMarker: "physical-A", Tenant: uuid('0'), Organization: uuid('1'), School: uuid('2'), Campus: uuid('3'), Timezone: "Europe/Paris", Locale: "fr"}
}
func manifest() Manifest {
	return Manifest{Version: 1, SourceSHA256: strings.Repeat("a", 64), Schools: []Mapping{mapping("north", 0), mapping("west", 4)}}
}
func TestLegacyUpgradeManifestFingerprint(t *testing.T) {
	m := manifest()
	a, e := m.Fingerprint()
	if e != nil {
		t.Fatal(e)
	}
	m.Schools[0], m.Schools[1] = m.Schools[1], m.Schools[0]
	b, e := m.Fingerprint()
	if e != nil || b != a || m.Schools[0].LegacySchool != "west" {
		t.Fatal("unstable or mutating canonicalization", e)
	}
	m.Schools[0].Timezone = "America/Toronto"
	c, e := m.Fingerprint()
	if e != nil || a == c {
		t.Fatal("timezone did not bind the manifest", e)
	}
}
func TestLegacyUpgradeRejectsAmbiguousMapping(t *testing.T) {
	for name, corrupt := range map[string]func(*Manifest){
		"duplicate legacy":      func(m *Manifest) { m.Schools[1].LegacySchool = "north" },
		"merged tenant":         func(m *Manifest) { m.Schools[1].Tenant = m.Schools[0].Tenant },
		"reused school":         func(m *Manifest) { m.Schools[1].School = m.Schools[0].School },
		"pool collision":        func(m *Manifest) { m.Schools[1].SourcePool = "west" },
		"split marker":          func(m *Manifest) { m.Schools[1].SourceMarker = "physical-B" },
		"unknown timezone":      func(m *Manifest) { m.Schools[0].Timezone = "Local" },
		"missing timezone":      func(m *Manifest) { m.Schools[0].Timezone = "" },
		"unsupported UI locale": func(m *Manifest) { m.Schools[0].Locale = "zz" },
		"bad tenant":            func(m *Manifest) { m.Schools[0].Tenant = "north" },
		"unknown version":       func(m *Manifest) { m.Version++ },
		"empty schools":         func(m *Manifest) { m.Schools = nil },
		"invalid source digest": func(m *Manifest) { m.SourceSHA256 = strings.Repeat("z", 64) },
	} {
		t.Run(name, func(t *testing.T) {
			m := manifest()
			corrupt(&m)
			if _, e := m.Fingerprint(); e == nil {
				t.Fatal("invalid mapping accepted")
			}
		})
	}
}
func TestStrictManifestInput(t *testing.T) {
	raw, _ := json.Marshal(manifest())
	if _, e := ReadManifest(bytes.NewReader(raw)); e != nil {
		t.Fatal(e)
	}
	for _, raw := range []string{string(raw) + " {}", strings.Replace(string(raw), "{", "{\"password\":\"sensitive-detail\",", 1), strings.Replace(string(raw), "{", "{\"version\":99,", 1), strings.Replace(string(raw), "\"legacySchool\":", "\"legacySchool\":\"other\",\"legacySchool\":", 1)} {
		_, e := ReadManifest(strings.NewReader(raw))
		if e == nil || strings.Contains(e.Error(), "sensitive-detail") {
			t.Fatal("ambiguous input or leaked content")
		}
	}
}
