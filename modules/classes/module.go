// Package classes exposes the integration surface of the Classes and sections module.
package classes

import (
	"github.com/elcomware/edupilot/modules/classes/app/services"
	records "github.com/elcomware/edupilot/modules/classes/database"
	"github.com/elcomware/edupilot/modules/classes/database/migrations"
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
