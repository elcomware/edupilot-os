// Package examinations exposes the integration surface of the Examinations module.
package examinations

import (
	"github.com/elcomware/edupilot-os/modules/examinations/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/examinations/application"
	"github.com/elcomware/edupilot-os/modules/examinations/migrations"
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
