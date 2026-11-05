// Package classes exposes the integration surface of the Classes and sections module.
package classes

import (
	"github.com/elcomware/edupilot-os/modules/classes/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/classes/application"
	"github.com/elcomware/edupilot-os/modules/classes/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"

	"github.com/elcomware/edupilot-os/platform/database"
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
