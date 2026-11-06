// Package schoolapi composes school module handlers at the versioned REST boundary.
package schoolapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/elcomware/edupilot/app/facades"
	modules "github.com/elcomware/edupilot/bootstrap/modulecatalog"
	"github.com/elcomware/edupilot/modules/core/access"
	"github.com/elcomware/edupilot/modules/core/catalog"
	"github.com/elcomware/edupilot/modules/core/workspace"
	"github.com/goravel/framework/contracts/cache"
)

type moduleAPI struct {
	registry   *modules.Registry
	cache      cache.Cache
	catalogKey string
}

func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, code, message string) {
	respond(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func authorized(w http.ResponseWriter, r *http.Request, permission string) (access.Principal, bool) {
	p, ok := access.FromContext(r.Context())
	if !ok {
		problem(w, 401, "unauthenticated", "Sign in to continue.")
		return p, false
	}
	if !p.Allows(permission) {
		problem(w, 403, "forbidden", "Permission denied.")
		return p, false
	}
	return p, true
}
func visible(d catalog.Descriptor, p access.Principal) catalog.Descriptor {
	read, write := access.WorkspaceRead, access.WorkspaceWrite
	capabilities := make([]string, 0, len(d.Capabilities))
	for _, capability := range d.Capabilities {
		if capability == "self" || (capability == "list" && p.Allows(read)) || ((capability == "create" || capability == "update") && p.Allows(write)) {
			capabilities = append(capabilities, capability)
		}
	}

	d.Capabilities = capabilities
	return d
}
func (a moduleAPI) list(w http.ResponseWriter, r *http.Request) {
	p, ok := authorized(w, r, access.ModulesRead)
	if !ok {
		return
	}
	// The catalog contains only public labels. Its four cache variants come from
	// current grants; session tokens, school records and principals never enter it.
	build := func() ([]byte, error) {
		items := make([]catalog.Descriptor, 0, len(a.registry.Definitions))
		for _, d := range a.registry.Definitions {
			entry := visible(d.Descriptor, p)
			if len(entry.Capabilities) > 0 {
				items = append(items, entry)
			}
		}
		return json.Marshal(map[string]any{"items": items})
	}
	key := fmt.Sprintf("catalog:v1:%s:%t:%t", a.catalogKey, p.Allows(access.WorkspaceRead), p.Allows(access.WorkspaceWrite))
	var body []byte
	if a.cache != nil {
		if value, ok := a.cache.Get(key).(string); ok {
			body = []byte(value)
		}
	}
	if body == nil {
		var err error
		body, err = build()
		if err != nil {
			problem(w, 500, "catalog_error", "Unable to load the module directory.")
			return
		}
		if a.cache != nil {
			if err := a.cache.Put(key, string(body), 5*time.Minute); err != nil {
				facades.Log().WithContext(r.Context()).Warning("module catalog cache unavailable")
			}
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(body)
}
func (a moduleAPI) detail(w http.ResponseWriter, r *http.Request) {
	p, ok := authorized(w, r, access.ModulesRead)
	if !ok {
		return
	}
	d, ok := a.registry.Find(r.PathValue("key"))
	if !ok {
		problem(w, 404, "not_found", "Unknown module.")
		return
	}
	entry := visible(d.Descriptor, p)
	if len(entry.Capabilities) == 0 {
		problem(w, 403, "forbidden", "Permission denied.")
		return
	}
	respond(w, 200, entry)
}
func decode(w http.ResponseWriter, r *http.Request, out any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil || dec.Decode(new(any)) != io.EOF {
		problem(w, 400, "invalid_json", "Expected one JSON object with the supported fields.")
		return false
	}
	return true
}
func pageNumber(r *http.Request, key string, fallback int) (int, error) {
	v, ok := r.URL.Query()[key]
	if !ok {
		return fallback, nil
	}
	if len(v) != 1 {
		return 0, workspace.ErrInvalid
	}
	n, err := strconv.Atoi(v[0])
	if err != nil {
		return 0, workspace.ErrInvalid
	}
	return n, nil
}
func (a moduleAPI) entries(w http.ResponseWriter, r *http.Request) {
	permission := access.WorkspaceRead
	if r.Method != http.MethodGet {
		permission = access.WorkspaceWrite
	}
	p, ok := authorized(w, r, permission)
	if !ok {
		return
	}
	key := r.PathValue("key")
	svc, ok := a.registry.Entries(key)
	if !ok {
		problem(w, 404, "not_found", "No work list for this module.")
		return
	}
	var value any
	var err error
	status := 200
	switch r.Method {
	case http.MethodGet:
		limit, e1 := pageNumber(r, "limit", 25)
		offset, e2 := pageNumber(r, "offset", 0)
		if e1 != nil || e2 != nil {
			err = workspace.ErrInvalid
			break
		}
		var page workspace.Page
		page, err = svc.List(r.Context(), p.School, limit, offset)
		value = struct {
			Module string `json:"module"`
			workspace.Page
		}{key, page}
	case http.MethodPost:
		var form struct {
			Title string `json:"title"`
		}
		if !decode(w, r, &form) {
			return
		}
		value, err = svc.Create(r.Context(), p.School, form.Title)
		status = 201
	case http.MethodPatch:
		var form struct {
			Title   string `json:"title"`
			Status  string `json:"status"`
			Version int64  `json:"version"`
		}
		if !decode(w, r, &form) {
			return
		}
		value, err = svc.Update(r.Context(), p.School, workspace.Entry{ID: r.PathValue("id"), Title: form.Title, Status: form.Status, Version: form.Version})
	}
	if err == nil {
		respond(w, status, value)
		return
	}
	switch {
	case errors.Is(err, workspace.ErrInvalid):
		problem(w, 422, "invalid_input", "Check the entry fields and pagination.")
	case errors.Is(err, workspace.ErrMissing):
		problem(w, 404, "not_found", "Entry not found.")
	case errors.Is(err, workspace.ErrConflict):
		problem(w, 409, "conflict", "Someone changed this entry. Reload it before saving.")
	default:
		facades.Log().WithContext(r.Context()).Errorf("workspace request failed for module %s", key)
		problem(w, 503, "unavailable", "Workspace storage is unavailable.")
	}
}

func (a moduleAPI) entry(w http.ResponseWriter, r *http.Request) {
	p, ok := authorized(w, r, access.WorkspaceRead)
	if !ok {
		return
	}
	svc, ok := a.registry.Entries(r.PathValue("key"))
	if !ok {
		problem(w, 404, "not_found", "Unknown work list.")
		return
	}
	item, err := svc.Get(r.Context(), p.School, r.PathValue("id"))
	if errors.Is(err, workspace.ErrMissing) {
		problem(w, 404, "not_found", "Entry not found.")
		return
	}
	if err != nil {
		problem(w, 503, "unavailable", "Workspace storage is unavailable.")
		return
	}
	respond(w, 200, item)
}
