// Package library exposes the integration surface of the Library module.
package library

import (
	"github.com/elcomware/edupilot/modules/library/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/library/application"
	"github.com/elcomware/edupilot/modules/library/migrations"
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
