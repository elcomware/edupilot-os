// Package admissions exposes the integration surface of the Admissions module.
package admissions

import (
	"github.com/elcomware/edupilot-os/modules/admissions/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/admissions/application"
	"github.com/elcomware/edupilot-os/modules/admissions/migrations"
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
