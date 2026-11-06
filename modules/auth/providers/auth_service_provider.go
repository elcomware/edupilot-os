package providers

import (
	"github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/modules/core/database"
	framework "github.com/goravel/framework/contracts/auth"
	"github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/contracts/http"
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
func (*AuthServiceProvider) Boot(app foundation.Application) {
	app.MakeAuth().Extend("school-bearer", func(ctx http.Context, _ string, _ framework.UserProvider) (framework.GuardDriver, error) {
		return &services.Guard{Request: ctx.Request().Origin(), Resolve: func() (*services.Service, error) { return Sessions(app) }}, nil
	})
}

func Sessions(app foundation.Application) (*services.Service, error) {
	value, err := app.Make(sessionBinding{})
	if err != nil {
		return nil, err
	}
	return value.(*services.Service), nil
}
