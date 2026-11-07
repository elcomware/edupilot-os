package routes

import (
	"github.com/elcomware/edupilot/modules/tenancy/app/http/controllers"
	goravelhttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"
	"net/http"
)

const hierarchyPath = "/api/v1/tenancy/hierarchy"

func API(router route.Router, dispatch goravelhttp.HandlerFunc) { router.Get(hierarchyPath, dispatch) }
func HTTP(controller controllers.HierarchyController, next http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+hierarchyPath, controller.List)
	mux.Handle("/", next)
	return mux
}
