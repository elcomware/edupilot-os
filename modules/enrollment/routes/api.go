package routes

import "net/http"

// Register mounts the collection and individual resources behind the host's
// authenticated school middleware.
func Register(mux *http.ServeMux, controller http.Handler) {
	const collection = "/api/v1/enrollment/placements"
	mux.Handle(collection, controller)
	mux.Handle(collection+"/", controller)
}
