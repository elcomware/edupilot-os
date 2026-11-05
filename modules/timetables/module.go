// Package timetables exposes the integration surface of the Timetables module.
package timetables

import (
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/modules/timetables/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/timetables/application"
	"github.com/elcomware/edupilot-os/modules/timetables/migrations"

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
