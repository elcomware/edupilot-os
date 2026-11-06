package schoolapi

import (
	"context"
	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/modules"
	"github.com/elcomware/edupilot/modules/auth/app/http/middleware"
	"github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/modules/documents"
	documentsroutes "github.com/elcomware/edupilot/modules/documents/routes"
	enrollment "github.com/elcomware/edupilot/modules/enrollment/providers"
	enrollmentroutes "github.com/elcomware/edupilot/modules/enrollment/routes"
	"github.com/elcomware/edupilot/modules/students/providers"
	studentsroutes "github.com/elcomware/edupilot/modules/students/routes"
	"github.com/elcomware/edupilot/platform/database"
	"net/http"
	"time"
)

func New(db *database.Catalog, sessions *services.Service, enabled string) (http.Handler, error) {
	definitions, err := modules.Enabled(modules.Definitions(), enabled)
	if err != nil {
		return nil, err
	}
	registry, err := modules.NewRegistry(db, definitions)
	if err != nil {
		return nil, err
	}
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
	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/v1/modules", api.list)
	protected.HandleFunc("GET /api/v1/modules/{key}", api.detail)
	protected.HandleFunc("GET /api/v1/modules/{key}/workspace", api.entries)
	protected.HandleFunc("POST /api/v1/modules/{key}/workspace", api.entries)
	protected.HandleFunc("GET /api/v1/modules/{key}/workspace/{id}", api.entry)
	protected.HandleFunc("PATCH /api/v1/modules/{key}/workspace/{id}", api.entries)
	if _, ok := registry.Find("students"); ok {
		studentsroutes.Register(protected, students.HTTP)
	}
	if _, ok := registry.Find("enrollment"); ok {
		enrollmentroutes.Register(protected, enrollment.New(db, students.Directory))
	}
	if _, ok := registry.Find("documents"); ok {
		documentsroutes.Register(protected, documents.Blobs())
	}
	mux.Handle("/", middleware.Protect(sessions, protected))
	return mux, nil
}
