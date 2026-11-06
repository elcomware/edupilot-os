package studenthttp

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/elcomware/edupilot/modules/students/application"
	"github.com/elcomware/edupilot/modules/students/contracts"
	"github.com/elcomware/edupilot/modules/students/domain"
	"github.com/elcomware/edupilot/platform/identity"
)

const collection = "/api/v1/students"

type Handler struct{ service *application.Service }

func New(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	p, ok := identity.FromContext(r.Context())
	if !ok {
		problem(w, 401, "unauthenticated", "Sign in to access student records.")
		return
	}
	path := r.URL.Path
	single := strings.HasPrefix(path, collection+"/")
	id := strings.TrimPrefix(path, collection+"/")
	if (path != collection && !single) || (single && (id == "" || strings.Contains(id, "/"))) {
		problem(w, 404, "not_found", "Student endpoint not found.")
		return
	}
	method := r.Method
	allowed := "GET, POST"
	if single {
		allowed = "GET, PATCH, DELETE"
	}
	if !((method == http.MethodGet) || (!single && method == http.MethodPost) || (single && (method == http.MethodPatch || method == http.MethodDelete))) {
		w.Header().Set("Allow", allowed)
		problem(w, 405, "method_not_allowed", "Method not allowed.")
		return
	}
	permission := identity.StudentsRead
	if method != http.MethodGet {
		permission = identity.StudentsWrite
	}
	if err := p.Require(p.School, permission); err != nil {
		problem(w, 403, "forbidden", "Permission denied.")
		return
	}

	switch {
	case method == http.MethodGet && !single:
		q, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			problem(w, 400, "invalid_query", "Invalid query string.")
			return
		}
		limit, ok := number(q, "limit", application.DefaultLimit)
		if !ok {
			problem(w, 400, "invalid_query", "limit must be an integer.")
			return
		}
		offset, ok := number(q, "offset", 0)
		if !ok {
			problem(w, 400, "invalid_query", "offset must be an integer.")
			return
		}
		page, err := h.service.List(r.Context(), p.School, limit, offset)
		if err != nil {
			failure(w, r, err)
			return
		}
		write(w, 200, page)
	case method == http.MethodGet:
		s, err := h.service.Lookup(r.Context(), p.School, id)
		if err != nil {
			failure(w, r, err)
			return
		}
		write(w, 200, s)
	case method == http.MethodPost:
		var in struct {
			GivenName  string `json:"givenName"`
			FamilyName string `json:"familyName"`
		}
		if !read(w, r, &in) {
			return
		}
		s, err := h.service.Register(r.Context(), p.School, p.ID, in.GivenName, in.FamilyName)
		if err != nil {
			failure(w, r, err)
			return
		}
		w.Header().Set("Location", collection+"/"+s.ID)
		write(w, 201, s)
	case method == http.MethodPatch:
		var in struct {
			GivenName  string `json:"givenName"`
			FamilyName string `json:"familyName"`
			Version    int64  `json:"version"`
		}
		if !read(w, r, &in) {
			return
		}
		s, err := h.service.Update(r.Context(), p.School, p.ID, id, in.Version, in.GivenName, in.FamilyName)
		if err != nil {
			failure(w, r, err)
			return
		}
		write(w, 200, s)
	case method == http.MethodDelete:
		q, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil || len(q["version"]) != 1 {
			problem(w, 400, "invalid_query", "Supply one integer version.")
			return
		}
		version, err := strconv.ParseInt(q.Get("version"), 10, 64)
		if err != nil {
			problem(w, 400, "invalid_query", "version must be an integer.")
			return
		}
		err = h.service.Archive(r.Context(), p.School, p.ID, id, version)
		if err != nil {
			failure(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func number(q url.Values, key string, def int) (int, bool) {
	values, ok := q[key]
	if !ok {
		return def, true
	}
	if len(values) != 1 {
		return 0, false
	}
	n, err := strconv.Atoi(values[0])
	return n, err == nil
}

func read[T any](w http.ResponseWriter, r *http.Request, dst *T) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var in *T
	err := dec.Decode(&in)
	if err == nil && in != nil {
		var extra json.RawMessage
		err = dec.Decode(&extra)
		if err == io.EOF {
			*dst = *in
			return true
		}
	}
	var limit *http.MaxBytesError
	if errors.As(err, &limit) {
		problem(w, 413, "request_too_large", "Student request exceeds 16 KiB.")
		return false
	}
	problem(w, 400, "invalid_json", "Supply one JSON object with the supported fields.")
	return false
}

func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func problem(w http.ResponseWriter, status int, code, message string) {
	write(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func failure(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalid):
		problem(w, 422, "invalid_student", err.Error())
	case errors.Is(err, contracts.ErrNotFound), errors.Is(err, domain.ErrNotFound):
		problem(w, 404, "not_found", "Student not found.")
	case errors.Is(err, domain.ErrConflict):
		problem(w, 409, "version_conflict", "The student record changed. Reload its current version.")
	case errors.Is(err, domain.ErrArchived):
		problem(w, 409, "student_archived", "This student record has been archived.")
	default:
		slog.ErrorContext(r.Context(), "student operation failed", "method", r.Method, "error", err)
		problem(w, 500, "internal_error", "Could not complete the student operation.")
	}
}
