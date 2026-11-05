package routes

import (
	"github.com/elcomware/edupilot-os/app/facades"
	"github.com/goravel/framework/contracts/http"
)

func Web() {
	facades.Route().Get("/", func(ctx http.Context) http.Response {
		return ctx.Response().Success().Json(http.Json{"application": "EduPilot OS", "api": "/api/v1", "health": "/api/v1/health"})
	})
	facades.Route().Static("public", "./public")
}
