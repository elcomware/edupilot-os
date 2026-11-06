package tenantupgrade

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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
	Documents SourceDocumentEvidence `json:"documents,omitzero"`
}

// SourceDocumentEvidence optionally binds an independently verified document
// inventory to the SQL inventory without disclosing object keys or contents.
type SourceDocumentEvidence struct {
	Present          bool   `json:"-"`
	Version          int    `json:"version"`
	Schools          int    `json:"schools"`
	References       int64  `json:"references,omitempty"`
	Documents        int64  `json:"documents"`
	ContentBytes     int64  `json:"contentBytes"`
	ReferencesSHA256 string `json:"referencesSha256,omitempty"`
	InventorySHA256  string `json:"inventorySha256,omitempty"`
	SHA256           string `json:"sha256"`
}

// Version 2 records an explicit zero when no objects are referenced. Version
// 1 predates that field, and the original strict reader rejects its presence.
func (d SourceDocumentEvidence) MarshalJSON() ([]byte, error) {
	type fields SourceDocumentEvidence
	var count *int64
	if d.Version == 2 || d.References != 0 {
		count = &d.References
	}
	return json.Marshal(struct {
		fields
		References *int64 `json:"references,omitempty"`
	}{fields(d), count})
}

type sourceSelectionWire struct {
	Version   int                         `json:"version"`
	Schools   []string                    `json:"schools"`
	Documents *sourceDocumentEvidenceWire `json:"documents,omitempty"`
}

type sourceDocumentEvidenceWire struct {
	Version          *int    `json:"version"`
	Schools          *int    `json:"schools"`
	References       *int64  `json:"references,omitempty"`
	Documents        *int64  `json:"documents"`
	ContentBytes     *int64  `json:"contentBytes"`
	ReferencesSHA256 *string `json:"referencesSha256,omitempty"`
	InventorySHA256  *string `json:"inventorySha256,omitempty"`
	SHA256           *string `json:"sha256"`
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
		documents, err := wire.Documents.evidence(wire.Version)
		if err != nil {
			return SourceSelection{}, err
		}
		selection.Documents = documents
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
			if err := sourceSelectionObject(d, []string{"version", "schools", "references", "documents", "contentBytes", "referencesSha256", "inventorySha256", "sha256"}, false); err != nil {
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
	return nil
}

func (w sourceDocumentEvidenceWire) evidence(selectionVersion int) (SourceDocumentEvidence, error) {
	if w.Version == nil || w.Schools == nil || w.Documents == nil || w.ContentBytes == nil || w.SHA256 == nil {
		return SourceDocumentEvidence{}, errSourceSelection
	}
	result := SourceDocumentEvidence{
		Present: true, Version: *w.Version, Schools: *w.Schools, Documents: *w.Documents,
		ContentBytes: *w.ContentBytes, SHA256: *w.SHA256,
	}
	switch selectionVersion {
	case 1:
		if result.Version != 1 || w.References != nil || w.ReferencesSHA256 != nil || w.InventorySHA256 != nil {
			return SourceDocumentEvidence{}, errSourceSelection
		}
	case 2:
		if result.Version != 2 || w.References == nil || w.ReferencesSHA256 == nil || w.InventorySHA256 == nil {
			return SourceDocumentEvidence{}, errSourceSelection
		}
		result.References = *w.References
		result.ReferencesSHA256 = *w.ReferencesSHA256
		result.InventorySHA256 = *w.InventorySHA256
	default:
		return SourceDocumentEvidence{}, errSourceSelection
	}
	return result, nil
}

func (s SourceSelection) Validate() error {
	if (s.Version != 1 && s.Version != 2) || len(s.Schools) == 0 || len(s.Schools) > maxSchools {
		return errSourceSelection
	}
	for i, school := range s.Schools {
		if !legacySlug.MatchString(school) || (i > 0 && strings.Compare(s.Schools[i-1], school) >= 0) {
			return errSourceSelection
		}
	}
	if s.Documents.Present {
		d := s.Documents
		if d.Schools != len(s.Schools) || d.Documents < 0 || d.ContentBytes < 0 || !sourceDigest.MatchString(d.SHA256) {
			return errSourceSelection
		}
		if s.Version == 1 {
			if d.Version != 1 || d.References != 0 || d.ReferencesSHA256 != "" || d.InventorySHA256 != "" {
				return errSourceSelection
			}
		} else {
			reconciled, err := ReconciledDocumentSHA256(d.Schools, d.References, d.Documents, d.ContentBytes, d.ReferencesSHA256, d.InventorySHA256)
			if d.Version != 2 || err != nil || reconciled != d.SHA256 {
				return errSourceSelection
			}
		}
	} else if s.Version == 2 || s.Documents.Version != 0 || s.Documents.Schools != 0 || s.Documents.References != 0 || s.Documents.Documents != 0 || s.Documents.ContentBytes != 0 || s.Documents.ReferencesSHA256 != "" || s.Documents.InventorySHA256 != "" || s.Documents.SHA256 != "" {
		return errSourceSelection
	}
	return nil
}

// ReconciledDocumentSHA256 is shared by the document verifier and source
// selection validator so retained evidence cannot claim inconsistent counts or
// component digests.
func ReconciledDocumentSHA256(schools int, references, documents, contentBytes int64, referencesSHA256, inventorySHA256 string) (string, error) {
	if schools <= 0 || references < 0 || references != documents || contentBytes < 0 ||
		!sourceDigest.MatchString(referencesSHA256) || !sourceDigest.MatchString(inventorySHA256) {
		return "", errSourceSelection
	}
	digest := sha256.New()
	writeFrame(digest, []byte("EDUPILOT-RECONCILED-LOCAL-DOCUMENT-INVENTORY/2"))
	for _, count := range []uint64{uint64(schools), uint64(references), uint64(documents), uint64(contentBytes)} {
		var raw [8]byte
		binary.BigEndian.PutUint64(raw[:], count)
		digest.Write(raw[:])
	}
	writeFrame(digest, []byte(referencesSHA256))
	writeFrame(digest, []byte(inventorySHA256))
	return hex.EncodeToString(digest.Sum(nil)), nil
}
