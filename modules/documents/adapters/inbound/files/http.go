// Package files serves the Documents module's private uploads and downloads.
package files

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/elcomware/edupilot/modules/documents/application"
	"github.com/elcomware/edupilot/platform/identity"
)

const path = "/api/v1/documents/files"

type handler struct {
	files application.Files
	work  chan struct{}
}

func New(files application.Files) http.Handler { return &handler{files, make(chan struct{}, 8)} }

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	p, ok := identity.FromContext(r.Context())
	if !ok {
		fail(w, 401, "unauthenticated", "Sign in to access files.")
		return
	}
	scope := r.Header.Values("X-School-ID")
	if len(scope) > 0 && (len(scope) != 1 || scope[0] != p.School) {
		fail(w, 403, "forbidden", "Access denied for this school.")
		return
	}

	upload := r.URL.Path == path
	id, download := strings.CutPrefix(r.URL.Path, path+"/")
	if !upload && (!download || id == "" || strings.Contains(id, "/")) {
		fail(w, 404, "not_found", "File not found.")
		return
	}
	method, permission := http.MethodGet, identity.DocumentsRead
	if upload {
		method, permission = http.MethodPost, identity.DocumentsWrite
	}
	if r.Method != method {
		w.Header().Set("Allow", method)
		fail(w, 405, "method_not_allowed", "Method not allowed.")
		return
	}
	if p.Require(p.School, permission) != nil {
		fail(w, 403, "forbidden", "Access denied for this action.")
		return
	}
	select {
	case h.work <- struct{}{}:
		defer func() { <-h.work }()
	default:
		w.Header().Set("Retry-After", "1")
		fail(w, 503, "busy", "Document storage is busy.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	if upload {
		h.upload(ctx, w, r)
		return
	}
	file, err := h.files.Download(ctx, id)
	if err != nil {
		handleError(w, err)
		return
	}
	// id is a canonical generated UUID and Extension comes from a static allowlist.
	w.Header().Set("Content-Type", file.MediaType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+id+"."+file.Extension+`"`)
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	w.Header().Set("Content-Length", strconv.Itoa(len(file.Content)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(file.Content)
}

func (h *handler) upload(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > application.MaxFileSize {
		handleError(w, application.ErrFileTooLarge)
		return
	}
	if len(r.Header.Values("Content-Type")) != 1 {
		handleError(w, application.ErrFileType)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, application.MaxFileSize)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		var limit *http.MaxBytesError
		if errors.As(err, &limit) {
			handleError(w, application.ErrFileTooLarge)
		} else {
			fail(w, 400, "invalid_request", "Could not read the file.")
		}
		return
	}
	id, err := h.files.Upload(ctx, r.Header.Get("Content-Type"), data)
	if err != nil {
		handleError(w, err)
		return
	}
	w.Header().Set("Location", path+"/"+id)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func fail(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, identity.ErrUnauthenticated):
		fail(w, 401, "unauthenticated", "Sign in to access files.")
	case errors.Is(err, identity.ErrForbidden):
		fail(w, 403, "forbidden", "Access denied for this action.")
	case errors.Is(err, application.ErrFileMissing):
		fail(w, 404, "not_found", "File not found.")
	case errors.Is(err, application.ErrFileTooLarge):
		fail(w, 413, "file_too_large", "Files must be at most 8 MiB.")
	case errors.Is(err, application.ErrFileEmpty):
		fail(w, 400, "empty_file", "The file is empty.")
	case errors.Is(err, application.ErrFileType):
		fail(w, 415, "unsupported_media_type", "Choose a PDF, PNG, JPEG, or UTF-8 text file with a matching Content-Type.")
	default:
		fail(w, 503, "storage_unavailable", "Document storage is unavailable.")
	}
}
