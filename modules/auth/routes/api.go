// Package routes owns the Auth endpoints, including the stable session URL
// used by the web, desktop and mobile clients.
package routes

import (
	"github.com/elcomware/edupilot/modules/auth/app/http/controllers"
	goravelhttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"
	"net/http"
)

const sessionPath = "/api/v1/session"
const currentUserPath = "/api/v1/auth/me"

// API exposes named endpoints to Goravel while retaining lazy database startup.
func API(router route.Router, dispatch goravelhttp.HandlerFunc) {
	router.Get(sessionPath, dispatch)
	router.Post(sessionPath, dispatch)
	router.Delete(sessionPath, dispatch)
	router.Get(currentUserPath, dispatch)
}

// HTTP mounts Auth ahead of the lazy school API. All HTTP routes use the same
// controller as the gRPC service, including its process-wide sign-in limiter.
func HTTP(auth *controllers.AuthController, schoolAPI http.Handler) http.Handler {
	mux := http.NewServeMux()
	registerWeb(mux, auth)
	mux.HandleFunc(sessionPath, auth.Session)
	mux.HandleFunc("GET "+currentUserPath, auth.CurrentUser)
	mux.Handle("/", schoolAPI)
	return mux
}
