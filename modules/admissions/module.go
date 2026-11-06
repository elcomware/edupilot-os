// Package admissions exposes the integration surface of the Admissions module.
package admissions

import (
	"github.com/elcomware/edupilot/modules/admissions/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/admissions/application"
	"github.com/elcomware/edupilot/modules/admissions/migrations"
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
