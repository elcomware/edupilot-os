// Package staff exposes the integration surface of the Staff module.
package staff

import (
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/modules/staff/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/staff/application"
	"github.com/elcomware/edupilot-os/modules/staff/migrations"

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
