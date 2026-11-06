package bootstrap

import (
	"github.com/elcomware/edupilot-os/database/edumigrations"
	"github.com/elcomware/edupilot-os/modules"
	documents "github.com/elcomware/edupilot-os/modules/documents/migrations"
	enrollment "github.com/elcomware/edupilot-os/modules/enrollment/migrations"
	students "github.com/elcomware/edupilot-os/modules/students/migrations"
	tenancy "github.com/elcomware/edupilot-os/modules/tenancy/migrations"
	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/elcomware/edupilot-os/platform/identity"
	"github.com/elcomware/edupilot-os/platform/outbox"
	"github.com/elcomware/edupilot-os/platform/workspace/sqlstore"
)

// FoundationPlans adds empty, versioned tables alongside the starter. The
// normal registry applies ControlOnly plans once and data plans on every pool.
func FoundationPlans() []edumigrations.Plan {
	plans := []edumigrations.Plan{
		{ID: "20261106010001_control", ControlOnly: true, ForwardOnly: true, DriverUp: map[string][]string{"sqlite": database.FoundationControlDDL("sqlite"), "postgres": database.FoundationControlDDL("postgres")}},
		{ID: "20261106010002_identity", ControlOnly: true, ForwardOnly: true, Up: identity.FoundationDDL()},
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
