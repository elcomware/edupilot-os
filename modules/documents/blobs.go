package documents

import (
	"net/http"
	"sync"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/modules/documents/app/http/controllers"
	"github.com/elcomware/edupilot/modules/documents/app/services"
	"github.com/elcomware/edupilot/modules/documents/app/services/filesystem"
)

// Blobs returns one handler for the collection and individual file routes.
// The composition root wraps it with the server's verified session middleware.
func Blobs() http.Handler {
	resolve := sync.OnceValues(func() (services.BlobStore, error) {
		return filesystem.Configured(func(k, fallback string) string { return facades.Config().EnvString(k, fallback) })
	})
	return controllers.New(services.Files{Resolve: resolve})
}
