package documents

import (
	"net/http"
	"sync"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/modules/documents/adapters/inbound/files"
	"github.com/elcomware/edupilot/modules/documents/adapters/outbound/blob"
	"github.com/elcomware/edupilot/modules/documents/application"
)

// Blobs returns one handler for the collection and individual file routes.
// The composition root wraps it with the server's verified session middleware.
func Blobs() http.Handler {
	resolve := sync.OnceValues(func() (application.BlobStore, error) {
		return blob.Configured(func(k, fallback string) string { return facades.Config().EnvString(k, fallback) })
	})
	return files.New(application.Files{Resolve: resolve})
}
