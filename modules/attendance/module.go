// Package attendance exposes the integration surface of the Attendance module.
package attendance

import (
	"github.com/elcomware/edupilot-os/modules/attendance/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/attendance/application"
	"github.com/elcomware/edupilot-os/modules/attendance/migrations"
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
