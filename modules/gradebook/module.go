// Package gradebook exposes the integration surface of the Gradebook module.
package gradebook

import (
	"github.com/elcomware/edupilot/modules/gradebook/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/gradebook/application"
	"github.com/elcomware/edupilot/modules/gradebook/migrations"
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
