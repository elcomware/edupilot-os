// Package learning exposes the integration surface of the Learning module.
package learning

import (
	"github.com/elcomware/edupilot/modules/learning/app/services"
	records "github.com/elcomware/edupilot/modules/learning/database"
	"github.com/elcomware/edupilot/modules/learning/database/migrations"
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
