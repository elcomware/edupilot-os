package providers

import (
	"sync"

	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/goravel/framework/contracts/foundation"
)

type CoreServiceProvider struct{}

func (*CoreServiceProvider) Boot(foundation.Application) {}
func (*CoreServiceProvider) Register(app foundation.Application) {
	// HTTP and gRPC can be the first request concurrently. Goravel caches a
	// singleton after its factory returns. Open one pool set and retry failures.
	var catalogMu sync.Mutex
	var catalog *database.Catalog
	app.Singleton(database.CatalogBinding, func(foundation.Application) (any, error) {
		catalogMu.Lock()
		defer catalogMu.Unlock()
		if catalog == nil {
			opened, err := database.OpenConfigured()
			if err != nil {
				return nil, err
			}
			catalog = opened
		}
		return catalog, nil
	})
}
