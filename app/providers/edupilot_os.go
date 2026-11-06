package providers

import (
	"github.com/elcomware/edupilot/app/schoolapi"
	auth "github.com/elcomware/edupilot/modules/auth/providers"
	"github.com/elcomware/edupilot/platform/database"
	"github.com/goravel/framework/contracts/foundation"
	"net/http"
	"sync"
)

type runtimeKey struct{}
type EduPilotOS struct{}

func (*EduPilotOS) Register(app foundation.Application) {
	// HTTP and gRPC can be the first request concurrently. Goravel caches a
	// singleton after its factory returns. Open one pool set and retry failures.
	var catalogMu sync.Mutex
	var catalog *database.Catalog
	app.Singleton(database.CatalogBinding, func(foundation.Application) (any, error) {
		catalogMu.Lock()
		defer catalogMu.Unlock()
		if catalog == nil {
			opened, err := database.OpenConfigured()
			if err != nil {
				return nil, err
			}
			catalog = opened
		}
		return catalog, nil
	})
	app.Singleton(runtimeKey{}, func(app foundation.Application) (any, error) {
		pools, err := app.Make(database.CatalogBinding)
		if err != nil {
			return nil, err
		}
		sessions, err := auth.Sessions(app)
		if err != nil {
			return nil, err
		}
		return schoolapi.New(pools.(*database.Catalog), sessions, app.MakeConfig().GetString("edupilot.enabled_modules"))
	})
}
func (*EduPilotOS) Boot(foundation.Application) {}

// Lazy resolution lets Artisan create keys or migrate a new database before the first API request.
func Handler(app foundation.Application) http.Handler {
	var mu sync.Mutex
	var handler http.Handler
	resolve := func() (http.Handler, error) {
		mu.Lock()
		defer mu.Unlock()
		if handler != nil {
			return handler, nil
		}
		raw, err := app.Make(runtimeKey{})
		if err != nil {
			return nil, err
		}
		handler = raw.(http.Handler)
		return handler, nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/health" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("{\"status\":\"ok\"}\n"))
			return
		}
		ready, err := resolve()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(503)
			_, _ = w.Write([]byte(`{"error":{"code":"unavailable","message":"School storage is unavailable."}}`))
			return
		}
		ready.ServeHTTP(w, r)
	})
}
