package routes

import (
	nethttp "net/http"

	"github.com/elcomware/edupilot/app/facades"
	authroutes "github.com/elcomware/edupilot/modules/auth/routes"
	"github.com/goravel/framework/contracts/http"
)

func Web(handler nethttp.Handler, assetsDir string) {
	authroutes.Web(facades.Route(), dispatch(handler))
	facades.Route().Static("/assets", assetsDir)
	facades.Route().Get("/", func(ctx http.Context) http.Response {
		return ctx.Response().Success().Json(http.Json{"application": "EduPilot OS", "api": "/api/v1", "health": "/api/v1/health"})
	})
}
