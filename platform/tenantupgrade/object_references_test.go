package tenantupgrade

import (
	"strings"
	"testing"
)

const validObjectReferences = `{"version":1,"schools":["east","west"],"objects":[{"school":"east","id":"00000001-0000-4000-8000-000000000001"},{"school":"west","id":"00000002-0000-4000-8000-000000000001"}]}`

func TestReadObjectReferencesRequiresCanonicalPreservedBaseline(t *testing.T) {
	references, err := ReadObjectReferences(strings.NewReader(validObjectReferences))
	if err != nil {
		t.Fatal(err)
	}
	digest, err := references.Fingerprint()
	if err != nil || len(digest) != 64 || references.Version != 1 || len(references.Schools) != 2 || len(references.Objects) != 2 {
		t.Fatalf("invalid preserved reference baseline: %#v %q %v", references, digest, err)
	}
	again, err := references.Fingerprint()
	if err != nil || again != digest {
		t.Fatal("reference fingerprint is not deterministic", err)
	}
}

func TestReadObjectReferencesRejectsAmbiguousOrNoncanonicalInput(t *testing.T) {
	cases := map[string]string{
		"unknown field":        strings.Replace(validObjectReferences, `"objects":`, `"private":"value","objects":`, 1),
		"duplicate field":      strings.Replace(validObjectReferences, `"version":1`, `"version":1,"version":1`, 1),
		"trailing value":       validObjectReferences + `{}`,
		"unsorted schools":     strings.Replace(validObjectReferences, `"east","west"`, `"west","east"`, 1),
		"duplicate school":     strings.Replace(validObjectReferences, `"east","west"`, `"east","east"`, 1),
		"foreign school":       strings.Replace(validObjectReferences, `"school":"west"`, `"school":"south"`, 1),
		"duplicate object":     strings.Replace(validObjectReferences, `"school":"west","id":"00000002`, `"school":"east","id":"00000001`, 1),
		"unsorted objects":     strings.Replace(validObjectReferences, `{"school":"east","id":"00000001-0000-4000-8000-000000000001"},{"school":"west","id":"00000002-0000-4000-8000-000000000001"}`, `{"school":"west","id":"00000002-0000-4000-8000-000000000001"},{"school":"east","id":"00000001-0000-4000-8000-000000000001"}`, 1),
		"noncanonical uuid":    strings.Replace(validObjectReferences, `00000001-0000-4000-8000-000000000001`, `00000001-0000-1000-8000-000000000001`, 1),
		"missing objects":      strings.Replace(validObjectReferences, `,"objects":[{"school":"east","id":"00000001-0000-4000-8000-000000000001"},{"school":"west","id":"00000002-0000-4000-8000-000000000001"}]`, ``, 1),
		"missing schools":      strings.Replace(validObjectReferences, `,"schools":["east","west"]`, ``, 1),
		"noncanonical version": strings.Replace(validObjectReferences, `"version":1`, `"version":2`, 1),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ReadObjectReferences(strings.NewReader(raw)); err == nil {
				t.Fatal("invalid reference baseline accepted")
			}
		})
	}
}
