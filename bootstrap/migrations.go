package bootstrap

import (
	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/database/migrations"
	"github.com/elcomware/edupilot/modules"
	auth "github.com/elcomware/edupilot/modules/auth/providers"
	"github.com/elcomware/edupilot/modules/enrollment/providers"
	students "github.com/elcomware/edupilot/modules/students/database/migrations"
	"github.com/elcomware/edupilot/platform/database"
	"github.com/goravel/framework/contracts/database/schema"
)

func Migrations() []schema.Migration {
	settings, err := database.ConfiguredSettings()
	if err != nil {
		panic(err)
	}
	plans := []edumigrations.Plan{
		edumigrations.PlacementPlan(), edumigrations.OutboxPlan(),
		auth.AccountsMigration(),
		{ID: "20260101000004_students", Up: students.SchemaUp, Down: students.SchemaDown},
		{ID: "20260101000005_enrollment_placements", Up: providers.SchemaUp, Down: providers.SchemaDown},
	}
	for _, d := range modules.Definitions() {
		if d.Migration != nil {
			plans = append(plans, d.Migration())
		}
	}
	plans = append(plans, FoundationPlans()...)
	return append([]schema.Migration{&migrations.M20210101000001CreateJobsTable{}}, edumigrations.Register(settings, plans...)...)
}
