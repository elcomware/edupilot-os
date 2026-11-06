package schoolapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/elcomware/edupilot/app/facades"
	modules "github.com/elcomware/edupilot/bootstrap/modulecatalog"
	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/goravel/framework/contracts/cache"
)

func New(db *database.Catalog, enabled string, resolve modules.ResolveWorkspace, cacheStore cache.Cache) (http.Handler, error) {
	definitions, err := modules.Enabled(modules.Definitions(), enabled)
	if err != nil {
		return nil, err
	}
	registry, err := modules.NewRegistry(resolve, definitions)
	if err != nil {
		return nil, err
	}

	keys := []string{}
	for _, d := range definitions {
		keys = append(keys, d.Key)
	}
	api := moduleAPI{registry: registry, cache: cacheStore, catalogKey: strings.Join(keys, ".")}
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
	mux.Handle("/", protected)
	return mux, nil
}
