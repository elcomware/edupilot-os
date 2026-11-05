package routes

import (
	"github.com/elcomware/edupilot-os/app/facades"
	"github.com/elcomware/edupilot-os/app/providers"
	"github.com/goravel/framework/contracts/http"
)

// Goravel owns the listener and middleware. A single net/http adapter lets the
// same module handlers run behind that listener and in feature specifications.
func API() {
	handler := providers.Handler(facades.App())
	facades.Route().Any("/api/v1/*path", func(ctx http.Context) http.Response {
		handler.ServeHTTP(ctx.Response().Writer(), ctx.Request().Origin())
		return nil
	})
}
