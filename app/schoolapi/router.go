package schoolapi

import (
	"context"
	"github.com/elcomware/edupilot-os/app/facades"
	"github.com/elcomware/edupilot-os/database/edumigrations"
	"github.com/elcomware/edupilot-os/modules"
	"github.com/elcomware/edupilot-os/modules/documents"
	"github.com/elcomware/edupilot-os/modules/enrollment/placements"
	"github.com/elcomware/edupilot-os/modules/students/providers"
	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/elcomware/edupilot-os/platform/identity"
	"net/http"
	"time"
)

func New(db *database.Catalog, enabled string) (http.Handler, error) {
	definitions, err := modules.Enabled(modules.Definitions(), enabled)
	if err != nil {
		return nil, err
	}
	registry, err := modules.NewRegistry(db, definitions)
	if err != nil {
		return nil, err
	}
	sessions := identity.New(&db.Control)
	students := providers.New(db)
	api := moduleAPI{registry}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := edumigrations.CheckReady(ctx, db, facades.Schema().Migrations()); err != nil {
			problem(w, 503, "unavailable", "School storage is unavailable.")
			return
		}
		respond(w, 200, map[string]string{"status": "ready"})
	})
	mux.Handle("/api/v1/session", identity.NewHandler(sessions))
	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/v1/modules", api.list)
	protected.HandleFunc("GET /api/v1/modules/{key}", api.detail)
	protected.HandleFunc("GET /api/v1/modules/{key}/workspace", api.entries)
	protected.HandleFunc("POST /api/v1/modules/{key}/workspace", api.entries)
	protected.HandleFunc("GET /api/v1/modules/{key}/workspace/{id}", api.entry)
	protected.HandleFunc("PATCH /api/v1/modules/{key}/workspace/{id}", api.entries)
	if _, ok := registry.Find("students"); ok {
		protected.Handle("/api/v1/students", students.HTTP)
		protected.Handle("/api/v1/students/", students.HTTP)
	}
	if _, ok := registry.Find("enrollment"); ok {
		handler := placements.New(db, students.Directory)
		protected.Handle("/api/v1/enrollment/placements", handler)
		protected.Handle("/api/v1/enrollment/placements/", handler)
	}
	if _, ok := registry.Find("documents"); ok {
		handler := documents.Blobs()
		protected.Handle("/api/v1/documents/files", handler)
		protected.Handle("/api/v1/documents/files/", handler)
	}
	mux.Handle("/", sessions.Protect(protected))
	return mux, nil
}
