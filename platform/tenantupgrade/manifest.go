// Package tenantupgrade defines the offline upgrade input and verification rules.
package tenantupgrade

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/elcomware/edupilot/platform/scope"
)

const maxManifestBytes = 4 << 20
const maxSchools = 4096

var (
	legacySlug   = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)
	poolName     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,79}$`)
	marker       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)
	sourceDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)
	errFormat    = errors.New("invalid version 1 upgrade manifest structure")
)

type Manifest struct {
	Version      int       `json:"version"`
	SourceSHA256 string    `json:"sourceSha256"`
	Schools      []Mapping `json:"schools"`
}

// Mapping explicitly allocates stable target IDs for one legacy school.
// SourcePool is a configured connection name, SourceMarker its physical identity.
type Mapping struct {
	LegacySchool string `json:"legacySchool"`
	SourcePool   string `json:"sourcePool"`
	SourceMarker string `json:"sourceMarker"`
	Tenant       string `json:"tenant"`
	Organization string `json:"organization"`
	School       string `json:"school"`
	Campus       string `json:"campus"`
	Timezone     string `json:"timezone"`
	Locale       string `json:"locale"`
}

func ReadManifest(input io.Reader) (Manifest, error) {
	data, err := io.ReadAll(io.LimitReader(input, maxManifestBytes+1))
	if err != nil {
		return Manifest{}, errors.New("could not read upgrade manifest")
	}
	if len(data) > maxManifestBytes {
		return Manifest{}, errors.New("upgrade manifest exceeds 4 MiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err = checkShape(decoder, 0); err != nil {
		return Manifest{}, err
	}
	if _, err = decoder.Token(); err != io.EOF {
		return Manifest{}, errFormat
	}
	var manifest Manifest
	if err = json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, errFormat
	}
	if err = manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// Reject duplicate fields before encoding/json can overwrite the first value.
// Only three structural levels are accepted: manifest, schools array, mapping.
func checkShape(d *json.Decoder, level int) error {
	token, err := d.Token()
	if err != nil {
		return errFormat
	}
	switch level {
	case 0, 2:
		if token != json.Delim('{') {
			return errFormat
		}
		allowed := []string{"version", "sourceSha256", "schools"}
		if level == 2 {
			allowed = []string{"legacySchool", "sourcePool", "sourceMarker", "tenant", "organization", "school", "campus", "timezone", "locale"}
		}
		seen := make(map[string]bool, len(allowed))
		for d.More() {
			field, err := d.Token()
			name, ok := field.(string)
			if err != nil || !ok || seen[name] || !slices.Contains(allowed, name) {
				return errFormat
			}
			seen[name] = true
			if level == 0 && name == "schools" {
				if err := checkShape(d, 1); err != nil {
					return err
				}
				continue
			}
			value, err := d.Token()
			if _, nested := value.(json.Delim); err != nil || nested || value == nil {
				return errFormat
			}
		}
		end, err := d.Token()
		if end != json.Delim('}') || err != nil || len(seen) != len(allowed) {
			return errFormat
		}
	case 1:
		if token != json.Delim('[') {
			return errFormat
		}
		for count := 0; d.More(); count++ {
			if count >= maxSchools {
				return errors.New("upgrade manifest exceeds school limit")
			}
			if err := checkShape(d, 2); err != nil {
				return err
			}
		}
		end, err := d.Token()
		if end != json.Delim(']') || err != nil {
			return errFormat
		}
	default:
		return errFormat
	}
	return nil
}

func (m Manifest) Validate() error {
	if m.Version != 1 || !sourceDigest.MatchString(m.SourceSHA256) || len(m.Schools) == 0 || len(m.Schools) > maxSchools {
		return errFormat
	}
	schools := map[string]bool{}
	ids := map[string]bool{}
	pools := map[string]string{}
	markers := map[string]string{}
	for i, s := range m.Schools {
		invalid := func(field string) error { return fmt.Errorf("mapping %d: invalid or conflicting %s", i+1, field) }
		if !legacySlug.MatchString(s.LegacySchool) || schools[s.LegacySchool] {
			return invalid("legacy school")
		}
		schools[s.LegacySchool] = true
		if !poolName.MatchString(s.SourcePool) || !marker.MatchString(s.SourceMarker) {
			return invalid("pool identity")
		}
		if other, exists := pools[s.SourcePool]; exists && other != s.SourceMarker {
			return invalid("pool marker")
		}
		if other, exists := markers[s.SourceMarker]; exists && other != s.SourcePool {
			return invalid("pool name")
		}
		pools[s.SourcePool] = s.SourceMarker
		markers[s.SourceMarker] = s.SourcePool
		if _, err := scope.Parse[scope.Tenant](s.Tenant); err != nil {
			return invalid("tenant ID")
		}
		if _, err := scope.Parse[scope.Organization](s.Organization); err != nil {
			return invalid("organization ID")
		}
		if _, err := scope.Parse[scope.School](s.School); err != nil {
			return invalid("school ID")
		}
		if _, err := scope.Parse[scope.Campus](s.Campus); err != nil {
			return invalid("campus ID")
		}
		for _, id := range []string{s.Tenant, s.Organization, s.School, s.Campus} {
			if ids[id] {
				return invalid("target identity")
			}
			ids[id] = true
		}
		if s.Timezone == "" || s.Timezone == "Local" {
			return invalid("IANA timezone")
		}
		if _, err := time.LoadLocation(s.Timezone); err != nil {
			return invalid("IANA timezone")
		}
		if s.Locale != "en" && s.Locale != "fr" {
			return invalid("initial UI locale")
		}
	}
	return nil
}

// Fingerprint binds every mapping and the source digest. Reordering equivalent
// mappings changes neither the fingerprint nor the caller's input slice.
func (m Manifest) Fingerprint() (string, error) {
	if err := m.Validate(); err != nil {
		return "", err
	}
	m.Schools = slices.Clone(m.Schools)
	slices.SortFunc(m.Schools, func(a, b Mapping) int { return strings.Compare(a.LegacySchool, b.LegacySchool) })
	canonical, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}
