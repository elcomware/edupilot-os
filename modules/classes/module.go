// Package classes exposes the integration surface of the Classes and sections module.
package classes

import (
	"github.com/elcomware/edupilot/modules/classes/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/classes/application"
	"github.com/elcomware/edupilot/modules/classes/migrations"
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
