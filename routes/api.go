package routes

import (
	nethttp "net/http"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/modules/auth/app/http/middleware"
	authroutes "github.com/elcomware/edupilot/modules/auth/routes"
	"github.com/goravel/framework/contracts/http"
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
	router.Get("/api/v1/health", serve)
	router.Get("/api/v1/ready", serve)
	protected := func(ctx http.Context) http.Response {
		middleware.ServeAuthenticated(ctx.Response().Writer(), ctx.Request().Origin(), handler, facades.Auth(ctx))
		return nil
	}
	router.Get("/api/v1/modules", protected)
	router.Any("/api/v1/modules/*path", protected)
	missing := func(ctx http.Context) http.Response {
		return ctx.Response().Json(404, http.Json{"error": http.Json{"code": "not_found", "message": "Unknown route."}})
	}
	router.Fallback(missing)
}
