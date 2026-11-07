package bootstrap

import (
	"os"
	"path/filepath"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/app/providers"
	"github.com/elcomware/edupilot/config"
	"github.com/elcomware/edupilot/modules/auth/app/http/controllers"
	"github.com/elcomware/edupilot/modules/auth/app/jobs"
	"github.com/elcomware/edupilot/modules/auth/app/services"
	auth "github.com/elcomware/edupilot/modules/auth/providers"
	authroutes "github.com/elcomware/edupilot/modules/auth/routes"
	"github.com/elcomware/edupilot/modules/core/access"
	contracts "github.com/elcomware/edupilot/modules/core/contracts/tenancy"
	tenancycontrollers "github.com/elcomware/edupilot/modules/tenancy/app/http/controllers"
	tenancyservices "github.com/elcomware/edupilot/modules/tenancy/app/services"
	tenancyproviders "github.com/elcomware/edupilot/modules/tenancy/providers"
	tenancyroutes "github.com/elcomware/edupilot/modules/tenancy/routes"
	"github.com/elcomware/edupilot/routes"
	contractsfoundation "github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/foundation"
)

func Boot() contractsfoundation.Application {
	return foundation.Setup().
		WithMigrations(Migrations).
		WithSeeders(auth.Seeders).
		WithJobs(jobs.Jobs).
		WithSchedule(jobs.Schedule).
		WithGrpcServerCredentials(config.GrpcServerCredentials).
		WithRouting(func() {
			app := facades.App()
			dist := app.BasePath("clients/web/dist")
			controller := controllers.New(func() (*services.Service, error) { return auth.Sessions(app) }, os.DirFS(dist))
			hierarchy := tenancycontrollers.HierarchyController{Authorizer: func() (access.TenantAuthorizer, error) { return auth.Sessions(app) }, Reader: func() (tenancyservices.HierarchyReader, error) { return tenancyproviders.HierarchyReader(app) }}
			directory := func() (contracts.Directory, error) {
				value, err := app.Make(contracts.DirectoryBinding)
				if err != nil {
					return nil, err
				}
				return value.(contracts.Directory), nil
			}
			handler := authroutes.HTTP(controller, tenancyroutes.HTTP(hierarchy, providers.Handler(app)), controller.Contexts(directory))
			routes.Web(handler, filepath.Join(dist, "assets"))
			routes.API(handler)
			routes.Grpc(controller)
		}).
		WithProviders(Providers).
		WithConfig(config.Boot).
		Create()
}
