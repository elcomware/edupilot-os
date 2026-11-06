package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"github.com/elcomware/edupilot/platform/media"
	"io"
	"mime"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/elcomware/edupilot/platform/access"
	"github.com/google/uuid"
)

const MaxFileSize = 8 << 20
const signature = "EDUPILOT-DOC/1\n"
const MaxRecordSize = MaxFileSize + len(signature) + sha256.Size

var (
	ErrFileMissing   = errors.New("document not found")
	ErrFileTooLarge  = errors.New("document too large")
	ErrFileEmpty     = errors.New("document is empty")
	ErrFileType      = errors.New("unsupported or mismatched document type")
	ErrFileIntegrity = errors.New("stored document failed integrity check")
)

// FileKey has no client-supplied path segments. Only the application constructs it.
type FileKey struct{ school, id string }

func (k FileKey) School() string      { return k.school }
func (k FileKey) ID() string          { return k.id }
func (k FileKey) StoragePath() string { return "schools/" + k.school + "/documents/" + k.id }

var safeSchool = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)

func key(school, id string) (FileKey, error) {
	u, err := uuid.Parse(id)
	if !safeSchool.MatchString(school) || err != nil || u.Version() != 4 || u.Variant() != uuid.RFC4122 || u.String() != id {
		return FileKey{}, ErrFileMissing
	}
	return FileKey{school, id}, nil
}

// BlobStore publishes one immutable, bounded record at a time.
type BlobStore interface {
	Put(context.Context, FileKey, []byte) error
	Get(context.Context, FileKey) ([]byte, error)
}

type Files struct{ Resolve func() (BlobStore, error) }

type Download struct {
	Content              []byte
	MediaType, Extension string
}

var extension = map[string]string{"text/plain": "txt", "application/pdf": "pdf", "image/png": "png", "image/jpeg": "jpg"}

func detect(data []byte) (string, error) {
	if len(data) == 0 {
		return "", ErrFileEmpty
	}
	if len(data) > MaxFileSize {
		return "", ErrFileTooLarge
	}
	mediaType, err := media.Detect(data)
	if err != nil || extension[mediaType] == "" {
		return "", ErrFileType
	}
	if mediaType == "text/plain" {
		if !utf8.Valid(data) {
			return "", ErrFileType
		}
		for _, b := range data {
			if b < 32 && b != '\n' && b != '\r' && b != '\t' {
				return "", ErrFileType
			}
		}
	}
	return mediaType, nil
}

func checksum(k FileKey, data []byte) []byte {
	h := sha256.New()
	_, _ = io.WriteString(h, k.StoragePath())
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(data)
	return h.Sum(nil)
}

// authorizedSchool prevents an internal caller from selecting a school by passing a string.
func authorizedSchool(ctx context.Context, permission string) (string, error) {
	principal, ok := access.FromContext(ctx)
	if !ok {
		return "", access.ErrUnauthenticated
	}
	if err := principal.Require(principal.School, permission); err != nil {
		return "", err
	}
	return principal.School, nil
}

func (f Files) Upload(ctx context.Context, declared string, data []byte) (string, error) {
	school, err := authorizedSchool(ctx, access.DocumentsWrite)
	if err != nil {
		return "", err
	}
	media, params, err := mime.ParseMediaType(declared)
	if err != nil || extension[media] == "" {
		return "", ErrFileType
	}
	if len(params) > 0 && !(media == "text/plain" && len(params) == 1 && strings.EqualFold(params["charset"], "utf-8")) {
		return "", ErrFileType
	}
	detected, err := detect(data)
	if err != nil {
		return "", err
	}
	if detected != media {
		return "", ErrFileType
	}
	u, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	k, err := key(school, u.String())
	if err != nil {
		return "", err
	}
	store, err := f.Resolve()
	if err != nil {
		return "", err
	}
	record := make([]byte, 0, len(data)+len(signature)+sha256.Size)
	record = append(record, signature...)
	record = append(record, checksum(k, data)...)
	record = append(record, data...)
	if err := store.Put(ctx, k, record); err != nil {
		return "", err
	}
	return k.ID(), nil
}

func (f Files) Download(ctx context.Context, id string) (Download, error) {
	school, err := authorizedSchool(ctx, access.DocumentsRead)
	if err != nil {
		return Download{}, err
	}
	k, err := key(school, id)
	if err != nil {
		return Download{}, err
	}
	store, err := f.Resolve()
	if err != nil {
		return Download{}, err
	}
	record, err := store.Get(ctx, k)
	if err != nil {
		return Download{}, err
	}
	return verifyRecord(k, record)
}

// verifyRecord is shared by authenticated downloads and offline backup inspection.
// It binds the original verified storage key to the envelope and supported bytes.
func verifyRecord(k FileKey, record []byte) (Download, error) {
	header := len(signature) + sha256.Size
	if len(record) <= header || len(record) > MaxRecordSize || !bytes.Equal(record[:len(signature)], []byte(signature)) {
		return Download{}, ErrFileIntegrity
	}
	data := record[header:]
	if !bytes.Equal(record[len(signature):header], checksum(k, data)) {
		return Download{}, ErrFileIntegrity
	}
	mediaType, err := detect(data)
	if err != nil {
		return Download{}, ErrFileIntegrity
	}
	return Download{Content: data, MediaType: mediaType, Extension: extension[mediaType]}, nil
}

// VerifyStoredRecord is a pure, bounded validator for trusted offline adapters.
// It returns the canonical original storage key and verified file bytes. This
// validates storage integrity only; it does not authenticate or authorize a user.
func VerifyStoredRecord(school, id string, input io.Reader) (FileKey, Download, error) {
	k, err := key(school, id)
	if err != nil {
		return FileKey{}, Download{}, ErrFileIntegrity
	}
	record, err := ReadRecord(input)
	if err != nil {
		return FileKey{}, Download{}, ErrFileIntegrity
	}
	file, err := verifyRecord(k, record)
	if err != nil {
		return FileKey{}, Download{}, err
	}
	return k, file, nil
}

// ReadRecord bounds reads even if the configured storage backend is damaged.
func ReadRecord(r io.Reader) ([]byte, error) {
	record, err := io.ReadAll(io.LimitReader(r, int64(MaxRecordSize)+1))
	if err != nil {
		return nil, err
	}
	if len(record) > MaxRecordSize {
		return nil, ErrFileIntegrity
	}
	return record, nil
}
