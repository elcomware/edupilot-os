package providers

import (
	"net/http"
	"sync"

	"github.com/elcomware/edupilot/app/schoolapi"
	"github.com/elcomware/edupilot/modules/core/contracts/modulekit"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/workspace"
	"github.com/goravel/framework/contracts/foundation"
)

type runtimeKey struct{}
type EduPilotOS struct{}

func (*EduPilotOS) Register(app foundation.Application) {
	app.Singleton(runtimeKey{}, func(app foundation.Application) (any, error) {
		pools, err := app.Make(database.CatalogBinding)
		if err != nil {
			return nil, err
		}
		return schoolapi.New(pools.(*database.Catalog), app.MakeConfig().GetString("edupilot.enabled_modules"), func(key string) (workspace.Service, error) {
			value, err := app.Make(modulekit.WorkspaceBinding(key))
			if err != nil {
				return workspace.Service{}, err
			}
			return value.(workspace.Service), nil
		}, app.MakeCache())
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
