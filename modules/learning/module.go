// Package learning exposes the integration surface of the Learning module.
package learning

import (
	"github.com/elcomware/edupilot-os/modules/learning/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/learning/application"
	"github.com/elcomware/edupilot-os/modules/learning/migrations"
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
