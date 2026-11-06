// Package guardians exposes the integration surface of the Guardians module.
package guardians

import (
	"github.com/elcomware/edupilot/modules/guardians/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/guardians/application"
	"github.com/elcomware/edupilot/modules/guardians/migrations"
	"github.com/elcomware/edupilot/modules/modulekit"

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
