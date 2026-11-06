// Package learning exposes the integration surface of the Learning module.
package learning

import (
	"github.com/elcomware/edupilot/modules/learning/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/learning/application"
	"github.com/elcomware/edupilot/modules/learning/migrations"
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
