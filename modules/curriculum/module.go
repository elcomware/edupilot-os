// Package curriculum exposes the integration surface of the Curriculum module.
package curriculum

import (
	"github.com/elcomware/edupilot-os/modules/curriculum/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/curriculum/application"
	"github.com/elcomware/edupilot-os/modules/curriculum/migrations"
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
