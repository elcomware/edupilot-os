// Package curriculum exposes the integration surface of the Curriculum module.
package curriculum

import (
	"github.com/elcomware/edupilot/modules/curriculum/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/curriculum/application"
	"github.com/elcomware/edupilot/modules/curriculum/migrations"
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
