// Package fees exposes the integration surface of the Fee policies module.
package fees

import (
	"github.com/elcomware/edupilot/modules/fees/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/fees/application"
	"github.com/elcomware/edupilot/modules/fees/migrations"
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
