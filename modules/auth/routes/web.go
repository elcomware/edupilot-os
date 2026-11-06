package routes

import (
	"github.com/elcomware/edupilot/modules/auth/app/http/controllers"
	goravelhttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"
	"net/http"
)

const loginPath = "/login"

func Web(router route.Router, dispatch goravelhttp.HandlerFunc) {
	router.Get(loginPath, dispatch)
}

func registerWeb(mux *http.ServeMux, auth *controllers.AuthController) {
	mux.HandleFunc("GET "+loginPath, auth.Session)
}
