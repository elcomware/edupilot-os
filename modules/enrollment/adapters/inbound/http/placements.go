package enrollmenthttp

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/elcomware/edupilot/modules/enrollment/application"
	"github.com/elcomware/edupilot/modules/enrollment/internal/domain"
	"github.com/elcomware/edupilot/platform/identity"
)

const collection = "/api/v1/enrollment/placements"

type handler struct{ service application.Placements }

func New(service application.Placements) http.Handler {
	h := handler{service}
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+collection, h.list)
	mux.HandleFunc("POST "+collection, h.record)
	mux.HandleFunc("PATCH "+collection+"/{id}", h.withdraw)
	return mux
}

type resource struct {
	ID           string `json:"id"`
	StudentID    string `json:"studentId"`
	AcademicYear string `json:"academicYear"`
	Grade        string `json:"grade"`
	Course       string `json:"course,omitempty"`
	Status       string `json:"status"`
	Version      int64  `json:"version"`
}

func view(p domain.Placement) resource {
	return resource{p.ID, p.StudentID, p.AcademicYear, p.Grade, p.Course, p.Status, p.Version}
}

func authorize(w http.ResponseWriter, r *http.Request, permissions ...string) (identity.Principal, bool) {
	w.Header().Set("Cache-Control", "no-store")
	p, ok := identity.FromContext(r.Context())
	if !ok {
		problem(w, 401, "unauthenticated", "Sign in to view placements.")
		return p, false
	}
	for _, perm := range permissions {
		if err := p.Require(p.School, perm); err != nil {
			problem(w, 403, "forbidden", "Permission denied.")
			return p, false
		}
	}
	return p, true
}

func (h handler) list(w http.ResponseWriter, r *http.Request) {
	p, ok := authorize(w, r, identity.StudentsRead, identity.WorkspaceRead)
	if !ok {
		return
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(q["studentId"]) > 1 {
		problem(w, 400, "invalid_query", "Invalid query string.")
		return
	}
	limit, ok := integer(q, "limit", 25)
	if !ok {
		problem(w, 400, "invalid_query", "limit must be one integer.")
		return
	}
	offset, ok := integer(q, "offset", 0)
	if !ok {
		problem(w, 400, "invalid_query", "offset must be one integer.")
		return
	}
	placements, total, err := h.service.List(r.Context(), p.School, q.Get("studentId"), limit, offset)
	if err != nil {
		failure(w, r, err)
		return
	}
	items := make([]resource, 0, len(placements))
	for _, p := range placements {
		items = append(items, view(p))
	}
	write(w, 200, struct {
		Items []resource `json:"items"`
		Total int64      `json:"total"`
	}{items, total})
}

func (h handler) record(w http.ResponseWriter, r *http.Request) {
	p, ok := authorize(w, r, identity.StudentsWrite, identity.WorkspaceWrite)
	if !ok {
		return
	}
	var input struct {
		StudentID    string `json:"studentId"`
		AcademicYear string `json:"academicYear"`
		Grade        string `json:"grade"`
		Course       string `json:"course"`
	}
	if !read(w, r, &input) {
		return
	}
	result, created, err := h.service.Record(r.Context(), p.School, p.ID, input.StudentID, input.AcademicYear, input.Grade, input.Course)
	if err != nil {
		failure(w, r, err)
		return
	}
	code := 200
	if created {
		code = 201
	}
	write(w, code, view(result))
}

func (h handler) withdraw(w http.ResponseWriter, r *http.Request) {
	p, ok := authorize(w, r, identity.StudentsWrite, identity.WorkspaceWrite)
	if !ok {
		return
	}
	var input struct {
		Status  string `json:"status"`
		Version int64  `json:"version"`
	}
	if !read(w, r, &input) {
		return
	}
	if input.Status != domain.WithdrawnPlacement {
		problem(w, 422, "invalid_placement", "status must be withdrawn.")
		return
	}
	result, err := h.service.Withdraw(r.Context(), p.School, p.ID, r.PathValue("id"), input.Version)
	if err != nil {
		failure(w, r, err)
		return
	}
	write(w, 200, view(result))
}
func integer(q url.Values, key string, fallback int) (int, bool) {
	values, present := q[key]
	if !present {
		return fallback, true
	}
	if len(values) != 1 {
		return 0, false
	}
	n, err := strconv.Atoi(values[0])
	return n, err == nil
}
func read[T any](w http.ResponseWriter, r *http.Request, destination *T) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var value *T
	err := decoder.Decode(&value)
	if err == nil && value != nil {
		var extra json.RawMessage
		err = decoder.Decode(&extra)
		if err == io.EOF {
			*destination = *value
			return true
		}
	}
	var max *http.MaxBytesError
	if errors.As(err, &max) {
		problem(w, 413, "request_too_large", "Request exceeds 16 KiB.")
		return false
	}
	problem(w, 400, "invalid_json", "Supply one JSON object with the supported fields.")
	return false
}
func write(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
func problem(w http.ResponseWriter, status int, code, message string) {
	write(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func failure(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalid):
		problem(w, 422, "invalid_placement", err.Error())
	case errors.Is(err, domain.ErrStudentMiss):
		problem(w, 404, "student_not_found", "Student not found.")
	case errors.Is(err, domain.ErrPlacementMiss):
		problem(w, 404, "placement_not_found", "Placement not found.")
	case errors.Is(err, domain.ErrStudentClosed):
		problem(w, 409, "student_archived", "This student has been archived.")
	case errors.Is(err, domain.ErrYearOccupied):
		problem(w, 409, "academic_year_conflict", "A different placement already exists for this academic year.")
	case errors.Is(err, domain.ErrWithdrawn):
		problem(w, 409, "placement_withdrawn", "This placement has been withdrawn.")
	case errors.Is(err, domain.ErrStale):
		problem(w, 409, "version_conflict", "The placement changed. Reload its current version.")
	default:
		slog.ErrorContext(r.Context(), "placement operation failed", "error", err)
		problem(w, 500, "internal_error", "Could not complete the placement operation.")
	}
}
