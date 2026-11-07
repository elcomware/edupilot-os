package bootstrap

import (
	"github.com/elcomware/edupilot/bootstrap/modulecatalog"
	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/database/migrations"
	auth "github.com/elcomware/edupilot/modules/auth/database/migrations"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/migration"
	tenancy "github.com/elcomware/edupilot/modules/tenancy/database/migrations"
	"github.com/goravel/framework/contracts/database/schema"
)

func Migrations() []schema.Migration {
	settings, err := database.ConfiguredSettings()
	if err != nil {
		panic(err)
	}
	var list []schema.Migration
	for _, node := range settings.Nodes() {
		control := node.Tenant == ""
		list = append(list, edumigrations.PoolBaseline(node, func(db migration.Schema) error { return tenancy.Placement(db, control) }))
		if control {
			list = append(list, edumigrations.OnPool(node, "02_accounts", auth.Accounts))
		}
		list = append(list, edumigrations.OnPool(node, "03_outbox", migration.Outbox), edumigrations.OnPool(node, "04_topology", tenancy.Topology))
		for _, m := range modulecatalog.Definitions() {
			if m.WorkspaceSchema != nil {
				list = append(list, edumigrations.OnPool(node, "05_workspace_"+m.Key, m.WorkspaceSchema))
			}
		}
		if control {
			list = append(list, edumigrations.OnPool(node, "06_reservations", tenancy.Reservations))
			list = append(list, edumigrations.OnPool(node, "07_activations", tenancy.Activations))
			list = append(list, edumigrations.OnPool(node, "08_admin_enrollments", auth.AdminEnrollments))
			list = append(list, edumigrations.OnPool(node, "09_tenant_grants", auth.TenantGrants))
		}
	}
	return append(list, &migrations.M20210101000001CreateJobsTable{})
}
