// Package enrollment exposes the integration surface of the Enrollment and transfers module.
package enrollment

import (
	"github.com/elcomware/edupilot-os/modules/enrollment/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/enrollment/application"
	"github.com/elcomware/edupilot-os/modules/enrollment/migrations"
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
