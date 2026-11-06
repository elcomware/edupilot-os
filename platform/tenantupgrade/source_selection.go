package tenantupgrade

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"
)

const maxSourceSelectionBytes = 4 << 20

var errSourceSelection = errors.New("invalid version 1 source selection structure")

// SourceSelection identifies only the historical school scopes to inventory.
// Target tenant identities deliberately do not belong in this pre-mapping input.
type SourceSelection struct {
	Version   int                    `json:"version"`
	Schools   []string               `json:"schools"`
	Documents SourceDocumentEvidence `json:"documents,omitempty"`
}

// SourceDocumentEvidence optionally binds an independently verified document
// inventory to the SQL inventory without disclosing object keys or contents.
type SourceDocumentEvidence struct {
	Present      bool   `json:"-"`
	Version      int    `json:"version"`
	Schools      int    `json:"schools"`
	Documents    int64  `json:"documents"`
	ContentBytes int64  `json:"contentBytes"`
	SHA256       string `json:"sha256"`
}

type sourceSelectionWire struct {
	Version   int                     `json:"version"`
	Schools   []string                `json:"schools"`
	Documents *SourceDocumentEvidence `json:"documents,omitempty"`
}

func ReadSourceSelection(input io.Reader) (SourceSelection, error) {
	data, err := io.ReadAll(io.LimitReader(input, maxSourceSelectionBytes+1))
	if err != nil {
		return SourceSelection{}, errors.New("could not read source selection")
	}
	if len(data) > maxSourceSelectionBytes {
		return SourceSelection{}, errors.New("source selection exceeds 4 MiB")
	}
	if err := validateSourceSelectionShape(data); err != nil {
		return SourceSelection{}, err
	}
	var wire sourceSelectionWire
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return SourceSelection{}, errSourceSelection
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return SourceSelection{}, errSourceSelection
	}
	selection := SourceSelection{Version: wire.Version, Schools: wire.Schools}
	if wire.Documents != nil {
		selection.Documents = *wire.Documents
		selection.Documents.Present = true
	}
	if err := selection.Validate(); err != nil {
		return SourceSelection{}, err
	}
	return selection, nil
}

func validateSourceSelectionShape(data []byte) error {
	var root any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil {
		return errSourceSelection
	}
	// Decode a second time token-by-token to reject duplicate fields that the
	// generic representation cannot retain.
	d := json.NewDecoder(bytes.NewReader(data))
	if err := sourceSelectionObject(d, []string{"version", "schools", "documents"}, true); err != nil {
		return err
	}
	return nil
}

func sourceSelectionObject(d *json.Decoder, allowed []string, root bool) error {
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return errSourceSelection
	}
	seen := map[string]bool{}
	for d.More() {
		raw, err := d.Token()
		name, ok := raw.(string)
		if err != nil || !ok || seen[name] || !slices.Contains(allowed, name) {
			return errSourceSelection
		}
		seen[name] = true
		if root && name == "schools" {
			start, err := d.Token()
			if err != nil || start != json.Delim('[') {
				return errSourceSelection
			}
			for count := 0; d.More(); count++ {
				if count >= maxSchools {
					return errors.New("source selection exceeds school limit")
				}
				value, err := d.Token()
				if _, ok := value.(string); err != nil || !ok {
					return errSourceSelection
				}
			}
			if end, err := d.Token(); err != nil || end != json.Delim(']') {
				return errSourceSelection
			}
			continue
		}
		if root && name == "documents" {
			if err := sourceSelectionObject(d, []string{"version", "schools", "documents", "contentBytes", "sha256"}, false); err != nil {
				return err
			}
			continue
		}
		value, err := d.Token()
		if _, nested := value.(json.Delim); err != nil || nested || value == nil {
			return errSourceSelection
		}
	}
	if end, err := d.Token(); err != nil || end != json.Delim('}') {
		return errSourceSelection
	}
	if !seen["version"] || (root && !seen["schools"]) {
		return errSourceSelection
	}
	if !root && len(seen) != len(allowed) {
		return errSourceSelection
	}
	return nil
}

func (s SourceSelection) Validate() error {
	if s.Version != 1 || len(s.Schools) == 0 || len(s.Schools) > maxSchools {
		return errSourceSelection
	}
	for i, school := range s.Schools {
		if !legacySlug.MatchString(school) || (i > 0 && strings.Compare(s.Schools[i-1], school) >= 0) {
			return errSourceSelection
		}
	}
	if s.Documents.Present {
		d := s.Documents
		if d.Version != 1 || d.Schools != len(s.Schools) || d.Documents < 0 || d.ContentBytes < 0 || !sourceDigest.MatchString(d.SHA256) {
			return errSourceSelection
		}
	} else if s.Documents.Version != 0 || s.Documents.Schools != 0 || s.Documents.Documents != 0 || s.Documents.ContentBytes != 0 || s.Documents.SHA256 != "" {
		return errSourceSelection
	}
	return nil
}
