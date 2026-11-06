// Package procurement exposes the integration surface of the Procurement module.
package procurement

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/procurement/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/procurement/application"
	"github.com/elcomware/edupilot/modules/procurement/migrations"

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
