package bootstrap

import (
	contractsfoundation "github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/foundation"

	"github.com/elcomware/edupilot/app/jobs"
	"github.com/elcomware/edupilot/config"
	"github.com/elcomware/edupilot/routes"
)

func Boot() contractsfoundation.Application {
	return foundation.Setup().
		WithMigrations(Migrations).
		WithJobs(jobs.Jobs).
		WithSchedule(jobs.Schedule).
		WithRouting(func() {
			routes.Web()
			routes.API()
			routes.Grpc()
		}).
		WithProviders(Providers).
		WithConfig(config.Boot).
		Create()
}
