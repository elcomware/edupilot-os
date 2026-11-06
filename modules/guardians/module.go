// Package guardians exposes the integration surface of the Guardians module.
package guardians

import (
	"github.com/elcomware/edupilot/modules/guardians/app/services"
	records "github.com/elcomware/edupilot/modules/guardians/database"
	"github.com/elcomware/edupilot/modules/guardians/database/migrations"
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
