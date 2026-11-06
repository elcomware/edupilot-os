package providers

import (
	"github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/platform/database"
	"github.com/goravel/framework/contracts/foundation"
)

type AuthServiceProvider struct{}
type sessionBinding struct{}

func (*AuthServiceProvider) Register(app foundation.Application) {
	app.Singleton(sessionBinding{}, func(app foundation.Application) (any, error) {
		catalog, err := app.Make(database.CatalogBinding)
		if err != nil {
			return nil, err
		}
		return services.New(&catalog.(*database.Catalog).Control), nil
	})
}
func (*AuthServiceProvider) Boot(foundation.Application) {}

func Sessions(app foundation.Application) (*services.Service, error) {
	value, err := app.Make(sessionBinding{})
	if err != nil {
		return nil, err
	}
	return value.(*services.Service), nil
}
