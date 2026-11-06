package bootstrap

import (
	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/modules"
	auth "github.com/elcomware/edupilot/modules/auth/providers"
	documents "github.com/elcomware/edupilot/modules/documents/database/migrations"
	enrollment "github.com/elcomware/edupilot/modules/enrollment/database/migrations"
	students "github.com/elcomware/edupilot/modules/students/database/migrations"
	tenancy "github.com/elcomware/edupilot/modules/tenancy/database/migrations"
	"github.com/elcomware/edupilot/platform/database"
	"github.com/elcomware/edupilot/platform/outbox"
	"github.com/elcomware/edupilot/platform/workspace/sqlstore"
)

// FoundationPlans adds empty, versioned tables alongside the starter. The
// normal registry applies ControlOnly plans once and data plans on every pool.
func FoundationPlans() []edumigrations.Plan {
	plans := []edumigrations.Plan{
		{ID: "20261106010001_control", ControlOnly: true, ForwardOnly: true, DriverUp: map[string][]string{"sqlite": database.FoundationControlDDL("sqlite"), "postgres": database.FoundationControlDDL("postgres")}},
		auth.FoundationMigration(),
		{ID: "20261106010003_topology", ForwardOnly: true, Up: tenancy.TopologyDDL()},
		{ID: "20261106010004_students", ForwardOnly: true, Up: students.OwnershipDDL()},
		{ID: "20261106010005_placements", ForwardOnly: true, Up: enrollment.OwnershipDDL()},
		{ID: "20261106010006_documents", ForwardOnly: true, Up: documents.MetadataDDL()},
		{ID: "20261106010007_events", ForwardOnly: true, Up: outbox.OwnershipDDL()},
	}
	for _, module := range modules.Definitions() {
		if module.Migration != nil {
			plans = append(plans, sqlstore.OwnershipPlan(module.Key))
		}
	}
	return plans
}
