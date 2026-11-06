// Package enrollment exposes the integration surface of the Enrollment and transfers module.
package enrollment

import (
	"github.com/elcomware/edupilot/modules/enrollment/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/enrollment/application"
	"github.com/elcomware/edupilot/modules/enrollment/migrations"
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
