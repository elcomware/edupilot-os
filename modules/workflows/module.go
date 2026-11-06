// Package workflows exposes the integration surface of the Workflows module.
package workflows

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/workflows/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/workflows/application"
	"github.com/elcomware/edupilot/modules/workflows/migrations"

	"github.com/elcomware/edupilot/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: Descriptor(),
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
