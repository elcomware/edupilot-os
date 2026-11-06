// Package workflows exposes the integration surface of the Workflows module.
package workflows

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/workflows/app/services"
	records "github.com/elcomware/edupilot/modules/workflows/database"
	"github.com/elcomware/edupilot/modules/workflows/database/migrations"

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
