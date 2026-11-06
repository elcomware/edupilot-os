// Package portal exposes the integration surface of the Family and student portal module.
package portal

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/portal/app/services"
	records "github.com/elcomware/edupilot/modules/portal/database"
	"github.com/elcomware/edupilot/modules/portal/database/migrations"

	"github.com/elcomware/edupilot/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: Descriptor(),
		Bind: func(db *database.Catalog) services.Entries {
			return services.Entries{Store: records.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
