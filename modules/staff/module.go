// Package staff exposes the integration surface of the Staff module.
package staff

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/staff/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/staff/application"
	"github.com/elcomware/edupilot/modules/staff/migrations"

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
