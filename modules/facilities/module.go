// Package facilities exposes the integration surface of the Facilities and assets module.
package facilities

import (
	"github.com/elcomware/edupilot/modules/facilities/app/services"
	records "github.com/elcomware/edupilot/modules/facilities/database"
	"github.com/elcomware/edupilot/modules/facilities/database/migrations"
	"github.com/elcomware/edupilot/modules/modulekit"

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
