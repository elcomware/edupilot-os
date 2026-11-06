// Package fees exposes the integration surface of the Fee policies module.
package fees

import (
	"github.com/elcomware/edupilot/modules/fees/app/services"
	records "github.com/elcomware/edupilot/modules/fees/database"
	"github.com/elcomware/edupilot/modules/fees/database/migrations"
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
