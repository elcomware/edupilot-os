package providers

import (
	"github.com/elcomware/edupilot/app/schoolapi"
	"github.com/elcomware/edupilot/platform/database"
	"github.com/goravel/framework/contracts/foundation"
	"net/http"
	"sync"
)

type runtimeKey struct{}
type EduPilotOS struct{}

func (*EduPilotOS) Register(app foundation.Application) {
	app.Singleton(runtimeKey{}, func(app foundation.Application) (any, error) {
		pools, err := database.OpenConfigured()
		if err != nil {
			return nil, err
		}
		return schoolapi.New(pools, app.MakeConfig().GetString("edupilot.enabled_modules"))
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
