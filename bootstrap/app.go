package bootstrap

import (
	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/app/providers"
	"github.com/elcomware/edupilot/modules/auth/app/http/controllers"
	"github.com/elcomware/edupilot/modules/auth/app/services"
	authroutes "github.com/elcomware/edupilot/modules/auth/routes"
	contractsfoundation "github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/foundation"
	"os"
	"path/filepath"

	"github.com/elcomware/edupilot/app/jobs"
	"github.com/elcomware/edupilot/config"
	auth "github.com/elcomware/edupilot/modules/auth/providers"
	"github.com/elcomware/edupilot/routes"
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
			handler := authroutes.HTTP(controller, providers.Handler(app))
			routes.Web(handler, filepath.Join(dist, "assets"))
			routes.API(handler)
			routes.Grpc(controller)
		}).
		WithProviders(Providers).
		WithConfig(config.Boot).
		Create()
}
