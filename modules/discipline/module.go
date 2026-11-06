// Package discipline exposes the integration surface of the Discipline and wellbeing module.
package discipline

import (
	"github.com/elcomware/edupilot/modules/discipline/app/services"
	records "github.com/elcomware/edupilot/modules/discipline/database"
	"github.com/elcomware/edupilot/modules/discipline/database/migrations"
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
