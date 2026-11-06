package routes

import (
	"github.com/elcomware/edupilot/app/facades"
	authroutes "github.com/elcomware/edupilot/modules/auth/routes"
	"github.com/goravel/framework/contracts/http"
	nethttp "net/http"
)

func dispatch(handler nethttp.Handler) http.HandlerFunc {
	return func(ctx http.Context) http.Response {
		w := ctx.Response().Writer()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		handler.ServeHTTP(w, ctx.Request().Origin())
		return nil
	}
}

func API(handler nethttp.Handler) {
	serve := dispatch(handler)
	router := facades.Route()
	authroutes.API(router, serve)
	// Static Auth routes take priority over these legacy dispatch paths.
	// Regular Goravel routes preserve net/http's default successful status.
	router.Any("/api/v1/", serve)
	router.Any("/api/v1/:resource", serve)
	router.Any("/api/v1/:resource/*path", serve)
}
