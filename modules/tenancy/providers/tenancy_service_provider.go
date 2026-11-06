package providers

import (
	"github.com/elcomware/edupilot/modules/core/contracts/modulekit"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/workspace"
	"github.com/elcomware/edupilot/modules/core/workspace/sqlstore"
	"github.com/elcomware/edupilot/modules/tenancy"
	"github.com/goravel/framework/contracts/foundation"
)

type TenancyServiceProvider struct{}

func (*TenancyServiceProvider) Boot(foundation.Application) {}
func (*TenancyServiceProvider) Register(app foundation.Application) {
	app.Singleton(modulekit.WorkspaceBinding(tenancy.Key), func(app foundation.Application) (any, error) {
		pools, err := app.Make(database.CatalogBinding)
		if err != nil {
			return nil, err
		}
		return workspace.Service{Store: sqlstore.New(pools.(*database.Catalog), tenancy.Key)}, nil
	})
}
