// Package blob implements private document storage.
package blob

import (
	"context"
	"errors"
	"io/fs"
	"os"

	"github.com/elcomware/edupilot-os/modules/documents/application"
	"github.com/google/uuid"
)

type local struct{ path string }

func (s local) directory(k application.FileKey, create bool) (*os.Root, error) {
	if create {
		if err := os.MkdirAll(s.path, 0700); err != nil {
			return nil, err
		}
	}
	dir, err := os.OpenRoot(s.path)
	if err != nil {
		return nil, err
	}
	for _, part := range []string{"schools", k.School(), "documents"} {
		if create {
			if err := dir.Mkdir(part, 0700); err != nil && !errors.Is(err, fs.ErrExist) {
				dir.Close()
				return nil, err
			}
		}
		info, err := dir.Lstat(part)
		if err != nil {
			dir.Close()
			return nil, err
		}
		if !info.IsDir() {
			dir.Close()
			return nil, errors.New("document directory must be a directory")
		}
		next, err := dir.OpenRoot(part)
		dir.Close()
		if err != nil {
			return nil, err
		}
		dir = next
	}
	return dir, nil
}

func (s local) Put(ctx context.Context, k application.FileKey, record []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dir, err := s.directory(k, true)
	if err != nil {
		return err
	}
	defer dir.Close()
	nonce, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	tmp := ".upload-" + nonce.String()
	f, err := dir.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer dir.Remove(tmp)
	if _, err = f.Write(record); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// A hard link publishes fully-written bytes and refuses to replace an existing ID.
	return dir.Link(tmp, k.ID())
}

func (s local) Get(ctx context.Context, k application.FileKey) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dir, err := s.directory(k, false)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, application.ErrFileMissing
	}
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	info, err := dir.Lstat(k.ID())
	if errors.Is(err, fs.ErrNotExist) {
		return nil, application.ErrFileMissing
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > int64(application.MaxRecordSize) {
		return nil, application.ErrFileIntegrity
	}
	f, err := dir.Open(k.ID())
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return application.ReadRecord(f)
}
