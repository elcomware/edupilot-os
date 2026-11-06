package tenantupgrade

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

const (
	maxObjectReferenceBytes = 4 << 20
	MaxObjectReferences     = 10_000
)

var errObjectReferences = errors.New("invalid version 1 object reference manifest")

// ObjectReferences is a private preserved baseline exported from the legacy
// records that refer to stored document objects. It must not be derived from
// storage enumeration, otherwise a missing object would also disappear here.
type ObjectReferences struct {
	Version int               `json:"version"`
	Schools []string          `json:"schools"`
	Objects []ObjectReference `json:"objects"`
}

type ObjectReference struct {
	School string `json:"school"`
	ID     string `json:"id"`
}

func ReadObjectReferences(input io.Reader) (ObjectReferences, error) {
	data, err := io.ReadAll(io.LimitReader(input, maxObjectReferenceBytes+1))
	if err != nil {
		return ObjectReferences{}, errors.New("could not read object reference manifest")
	}
	if len(data) > maxObjectReferenceBytes {
		return ObjectReferences{}, errors.New("object reference manifest exceeds 4 MiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := checkObjectReferenceShape(decoder); err != nil {
		return ObjectReferences{}, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return ObjectReferences{}, errObjectReferences
	}
	var references ObjectReferences
	if err := json.Unmarshal(data, &references); err != nil {
		return ObjectReferences{}, errObjectReferences
	}
	if err := references.Validate(); err != nil {
		return ObjectReferences{}, err
	}
	return references, nil
}

func checkObjectReferenceShape(decoder *json.Decoder) error {
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return errObjectReferences
	}
	seen := map[string]bool{}
	for decoder.More() {
		raw, err := decoder.Token()
		name, ok := raw.(string)
		if err != nil || !ok || seen[name] || !slices.Contains([]string{"version", "schools", "objects"}, name) {
			return errObjectReferences
		}
		seen[name] = true
		switch name {
		case "version":
			value, err := decoder.Token()
			if _, nested := value.(json.Delim); err != nil || nested || value == nil {
				return errObjectReferences
			}
		case "schools":
			if err := readReferenceSchools(decoder); err != nil {
				return err
			}
		case "objects":
			if err := readReferenceObjects(decoder); err != nil {
				return err
			}
		}
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim('}') || len(seen) != 3 {
		return errObjectReferences
	}
	return nil
}

func readReferenceSchools(decoder *json.Decoder) error {
	start, err := decoder.Token()
	if err != nil || start != json.Delim('[') {
		return errObjectReferences
	}
	for count := 0; decoder.More(); count++ {
		if count >= maxSchools {
			return errors.New("object reference manifest exceeds school limit")
		}
		value, err := decoder.Token()
		if _, ok := value.(string); err != nil || !ok {
			return errObjectReferences
		}
	}
	if end, err := decoder.Token(); err != nil || end != json.Delim(']') {
		return errObjectReferences
	}
	return nil
}

func readReferenceObjects(decoder *json.Decoder) error {
	start, err := decoder.Token()
	if err != nil || start != json.Delim('[') {
		return errObjectReferences
	}
	for count := 0; decoder.More(); count++ {
		if count >= MaxObjectReferences {
			return errors.New("object reference manifest exceeds object limit")
		}
		start, err := decoder.Token()
		if err != nil || start != json.Delim('{') {
			return errObjectReferences
		}
		seen := map[string]bool{}
		for decoder.More() {
			raw, err := decoder.Token()
			name, ok := raw.(string)
			if err != nil || !ok || seen[name] || !slices.Contains([]string{"school", "id"}, name) {
				return errObjectReferences
			}
			seen[name] = true
			value, err := decoder.Token()
			if _, ok := value.(string); err != nil || !ok {
				return errObjectReferences
			}
		}
		if end, err := decoder.Token(); err != nil || end != json.Delim('}') || len(seen) != 2 {
			return errObjectReferences
		}
	}
	if end, err := decoder.Token(); err != nil || end != json.Delim(']') {
		return errObjectReferences
	}
	return nil
}

func (r ObjectReferences) Validate() error {
	if r.Version != 1 || len(r.Schools) == 0 || len(r.Schools) > maxSchools || len(r.Objects) > MaxObjectReferences {
		return errObjectReferences
	}
	allowed := make(map[string]bool, len(r.Schools))
	for i, school := range r.Schools {
		if !legacySlug.MatchString(school) || (i > 0 && strings.Compare(r.Schools[i-1], school) >= 0) {
			return errObjectReferences
		}
		allowed[school] = true
	}
	for i, object := range r.Objects {
		parsed, err := uuid.Parse(object.ID)
		if !allowed[object.School] || err != nil || parsed.Version() != 4 || parsed.Variant() != uuid.RFC4122 || parsed.String() != object.ID {
			return errObjectReferences
		}
		if i > 0 {
			previous := r.Objects[i-1]
			if strings.Compare(previous.School, object.School) > 0 || (previous.School == object.School && strings.Compare(previous.ID, object.ID) >= 0) {
				return errObjectReferences
			}
		}
	}
	return nil
}

// Fingerprint binds the exact approved school membership and object-key set
// without exposing either in the reconciliation report.
func (r ObjectReferences) Fingerprint() (string, error) {
	if err := r.Validate(); err != nil {
		return "", err
	}
	hash := sha256.New()
	writeFrame(hash, []byte("EDUPILOT-OBJECT-REFERENCES/1"))
	writeFrame(hash, []byte(strconv.Itoa(len(r.Schools))))
	for _, school := range r.Schools {
		writeFrame(hash, []byte(school))
	}
	writeFrame(hash, []byte(strconv.Itoa(len(r.Objects))))
	for _, object := range r.Objects {
		writeFrame(hash, []byte(object.School))
		writeFrame(hash, []byte(object.ID))
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
