// Package hr exposes the integration surface of the Human resources module.
package hr

import (
	"github.com/elcomware/edupilot/modules/hr/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/hr/application"
	"github.com/elcomware/edupilot/modules/hr/migrations"
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
