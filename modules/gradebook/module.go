// Package gradebook exposes the integration surface of the Gradebook module.
package gradebook

import (
	"github.com/elcomware/edupilot/modules/gradebook/app/services"
	records "github.com/elcomware/edupilot/modules/gradebook/database"
	"github.com/elcomware/edupilot/modules/gradebook/database/migrations"
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
