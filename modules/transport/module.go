// Package transport exposes the integration surface of the Transport module.
package transport

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/transport/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/transport/application"
	"github.com/elcomware/edupilot/modules/transport/migrations"

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
