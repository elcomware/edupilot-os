package blob

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/elcomware/edupilot-os/modules/documents/application"
	"github.com/elcomware/edupilot-os/platform/tenantupgrade"
)

const (
	LocalInventoryVersion          = 1
	MaxInventoryDocuments          = 10_000
	MaxInventoryEntries            = 20_000
	MaxInventoryContentBytes int64 = 1 << 30
)

var (
	ErrLocalInventory       = errors.New("local document inventory is invalid or unavailable")
	ErrLocalInventoryLimit  = errors.New("local document inventory exceeds supported inspection limits")
	ErrLocalInventoryDigest = errors.New("local document inventory does not match expected digest")
)

// LocalInventory is a private aggregate for a frozen, immutable local backup.
// Schools counts approved manifest mappings, including those without documents.
// It is not a live-store completeness claim or evidence about an S3 bucket.
type LocalInventory struct {
	Version      int    `json:"version"`
	Schools      int    `json:"schools"`
	Documents    int    `json:"documents"`
	ContentBytes int64  `json:"contentBytes"`
	SHA256       string `json:"sha256"`
}

// ReconciledLocalInventory proves that the independently preserved reference
// set and the verified frozen byte inventory describe exactly the same objects.
// It exposes only counts and digests, never historical object keys.
type ReconciledLocalInventory struct {
	Version          int    `json:"version"`
	Schools          int    `json:"schools"`
	References       int    `json:"references"`
	Documents        int    `json:"documents"`
	ContentBytes     int64  `json:"contentBytes"`
	ReferencesSHA256 string `json:"referencesSha256"`
	InventorySHA256  string `json:"inventorySha256"`
	SHA256           string `json:"sha256"`
}

type localInspection struct {
	ctx      context.Context
	entries  int
	result   LocalInventory
	digest   hash.Hash
	expected map[tenantupgrade.ObjectReference]bool
}

// InventoryLocal inspects an explicitly existing private root without writing it.
// Only the exact starter layout is accepted. Every document is verified against
// its original path with the same envelope/bound/media rules as Download. The
// caller must provide a frozen backup and protect the resulting aggregate.
func InventoryLocal(ctx context.Context, path string, manifest tenantupgrade.Manifest, expectedSHA256 string) (LocalInventory, error) {
	return inventoryLocal(ctx, path, manifest, expectedSHA256, nil)
}

// ReconcileLocal requires an independently approved version 1 byte digest and
// an independently exported reference baseline. A reference list produced by
// enumerating the same storage root would not establish completeness.
func ReconcileLocal(ctx context.Context, path string, manifest tenantupgrade.Manifest, references tenantupgrade.ObjectReferences, expectedSHA256 string) (ReconciledLocalInventory, error) {
	if err := manifest.Validate(); err != nil || references.Validate() != nil || expectedSHA256 == "" {
		return ReconciledLocalInventory{}, ErrLocalInventory
	}
	schools := make([]string, len(manifest.Schools))
	for i, mapping := range manifest.Schools {
		schools[i] = mapping.LegacySchool
	}
	slices.Sort(schools)
	if !slices.Equal(schools, references.Schools) {
		return ReconciledLocalInventory{}, ErrLocalInventory
	}
	expected := make(map[tenantupgrade.ObjectReference]bool, len(references.Objects))
	for _, reference := range references.Objects {
		expected[reference] = true
	}
	inventory, err := inventoryLocal(ctx, path, manifest, expectedSHA256, expected)
	if err != nil {
		return ReconciledLocalInventory{}, err
	}
	referenceDigest, err := references.Fingerprint()
	if err != nil {
		return ReconciledLocalInventory{}, ErrLocalInventory
	}
	result := ReconciledLocalInventory{
		Version: 2, Schools: inventory.Schools, References: len(references.Objects), Documents: inventory.Documents,
		ContentBytes: inventory.ContentBytes, ReferencesSHA256: referenceDigest, InventorySHA256: inventory.SHA256,
	}
	result.SHA256, err = tenantupgrade.ReconciledDocumentSHA256(
		result.Schools, int64(result.References), int64(result.Documents), result.ContentBytes,
		result.ReferencesSHA256, result.InventorySHA256,
	)
	if err != nil {
		return ReconciledLocalInventory{}, ErrLocalInventory
	}
	return result, nil
}

func inventoryLocal(ctx context.Context, path string, manifest tenantupgrade.Manifest, expectedSHA256 string, expected map[tenantupgrade.ObjectReference]bool) (LocalInventory, error) {
	if err := ctx.Err(); err != nil {
		return LocalInventory{}, err
	}
	if err := manifest.Validate(); err != nil {
		return LocalInventory{}, ErrLocalInventory
	}
	if expectedSHA256 != "" {
		if len(expectedSHA256) != sha256.Size*2 {
			return LocalInventory{}, ErrLocalInventoryDigest
		}
		raw, err := hex.DecodeString(expectedSHA256)
		if err != nil || len(raw) != sha256.Size || strings.ToLower(expectedSHA256) != expectedSHA256 {
			return LocalInventory{}, ErrLocalInventoryDigest
		}
	}
	if path == "" {
		return LocalInventory{}, ErrLocalInventory
	}
	// Reject parent traversal before lexical cleaning: link/../backup can name
	// a different physical source from the cleaned path, or hide a missing part.
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return LocalInventory{}, ErrLocalInventory
		}
	}
	// Reject symlinked roots/components rather than silently changing the source.
	absolute, err := filepath.Abs(path)
	if err != nil {
		return LocalInventory{}, ErrLocalInventory
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil || resolved != absolute {
		return LocalInventory{}, ErrLocalInventory
	}
	info, err := os.Lstat(absolute)
	if err != nil || !privateDirectory(info) {
		return LocalInventory{}, ErrLocalInventory
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return LocalInventory{}, ErrLocalInventory
	}
	defer root.Close()
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return LocalInventory{}, ErrLocalInventory
	}
	scan := localInspection{ctx: ctx, result: LocalInventory{Version: LocalInventoryVersion, Schools: len(manifest.Schools)}, digest: sha256.New(), expected: expected}
	frame(scan.digest, []byte("EDUPILOT-LOCAL-DOCUMENT-INVENTORY/1"))
	legacySchools := make([]string, len(manifest.Schools))
	allowed := make(map[string]bool, len(manifest.Schools))
	for i, mapping := range manifest.Schools {
		legacySchools[i] = mapping.LegacySchool
		allowed[mapping.LegacySchool] = true
	}
	slices.Sort(legacySchools)
	frame(scan.digest, []byte("schools"))
	var schoolCount [8]byte
	binary.BigEndian.PutUint64(schoolCount[:], uint64(len(legacySchools)))
	scan.digest.Write(schoolCount[:])
	for _, school := range legacySchools {
		frame(scan.digest, []byte(school))
	}
	if err = scan.root(root, allowed); err != nil {
		return LocalInventory{}, err
	}
	if scan.expected != nil && len(scan.expected) != 0 {
		return LocalInventory{}, ErrLocalInventory
	}
	if err = ctx.Err(); err != nil {
		return LocalInventory{}, err
	}
	frame(scan.digest, []byte("totals"))
	for _, count := range []uint64{uint64(scan.result.Schools), uint64(scan.result.Documents), uint64(scan.result.ContentBytes)} {
		var raw [8]byte
		binary.BigEndian.PutUint64(raw[:], count)
		scan.digest.Write(raw[:])
	}
	scan.result.SHA256 = hex.EncodeToString(scan.digest.Sum(nil))
	if expectedSHA256 != "" && scan.result.SHA256 != expectedSHA256 {
		return LocalInventory{}, ErrLocalInventoryDigest
	}
	return scan.result, nil
}

func frame(h hash.Hash, value []byte) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	h.Write(size[:])
	h.Write(value)
}
func privateDirectory(info os.FileInfo) bool { return info.IsDir() && info.Mode().Perm()&0077 == 0 }
func directory(parent *os.Root, name string) (*os.Root, error) {
	info, err := parent.Lstat(name)
	if err != nil || !privateDirectory(info) {
		return nil, ErrLocalInventory
	}
	root, err := parent.OpenRoot(name)
	if err != nil {
		return nil, ErrLocalInventory
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		root.Close()
		return nil, ErrLocalInventory
	}
	return root, nil
}
func (s *localInspection) list(root *os.Root) ([]os.DirEntry, error) {
	if err := s.ctx.Err(); err != nil {
		return nil, err
	}
	file, err := root.Open(".")
	if err != nil {
		return nil, ErrLocalInventory
	}
	defer file.Close()
	remaining := MaxInventoryEntries - s.entries
	entries, err := file.ReadDir(remaining + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, ErrLocalInventory
	}
	if len(entries) > remaining {
		return nil, ErrLocalInventoryLimit
	}
	s.entries += len(entries)
	slices.SortFunc(entries, func(a, b os.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, nil
}
func (s *localInspection) root(root *os.Root, allowed map[string]bool) error {
	entries, err := s.list(root)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	if len(entries) != 1 || entries[0].Name() != "schools" {
		return ErrLocalInventory
	}
	schools, err := directory(root, "schools")
	if err != nil {
		return err
	}
	defer schools.Close()
	entries, err = s.list(schools)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !allowed[entry.Name()] {
			return ErrLocalInventory
		}
		if err = s.school(schools, entry.Name()); err != nil {
			return err
		}
	}
	return nil
}
func (s *localInspection) school(schools *os.Root, name string) error {
	root, err := directory(schools, name)
	if err != nil {
		return err
	}
	defer root.Close()
	entries, err := s.list(root)
	if err != nil {
		return err
	}
	if len(entries) != 1 || entries[0].Name() != "documents" {
		return ErrLocalInventory
	}
	documents, err := directory(root, "documents")
	if err != nil {
		return err
	}
	defer documents.Close()
	entries, err = s.list(documents)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err = s.document(documents, name, entry.Name()); err != nil {
			return err
		}
	}
	return nil
}
func (s *localInspection) document(root *os.Root, school, id string) error {
	if err := s.ctx.Err(); err != nil {
		return err
	}
	if s.result.Documents >= MaxInventoryDocuments {
		return ErrLocalInventoryLimit
	}
	info, err := root.Lstat(id)
	header := int64(application.MaxRecordSize - application.MaxFileSize)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() <= header || info.Size() > int64(application.MaxRecordSize) {
		return ErrLocalInventory
	}
	if info.Size()-header > MaxInventoryContentBytes-s.result.ContentBytes {
		return ErrLocalInventoryLimit
	}
	file, err := root.Open(id)
	if err != nil {
		return ErrLocalInventory
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return ErrLocalInventory
	}
	k, document, err := application.VerifyStoredRecord(school, id, file)
	if err != nil {
		return ErrLocalInventory
	}
	if s.expected != nil {
		reference := tenantupgrade.ObjectReference{School: school, ID: id}
		if !s.expected[reference] {
			return ErrLocalInventory
		}
		delete(s.expected, reference)
	}
	after, err := file.Stat()
	if err != nil || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
		return ErrLocalInventory
	}
	if int64(len(document.Content)) > MaxInventoryContentBytes-s.result.ContentBytes {
		return ErrLocalInventoryLimit
	}
	frame(s.digest, []byte("document"))
	frame(s.digest, []byte(k.StoragePath()))
	frame(s.digest, document.Content)
	s.result.Documents++
	s.result.ContentBytes += int64(len(document.Content))
	return nil
}
