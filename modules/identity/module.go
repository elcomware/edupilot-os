// Package identity exposes the integration surface of the Identity module.
package identity

import (
	"github.com/elcomware/edupilot/modules/identity/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/identity/application"
	"github.com/elcomware/edupilot/modules/identity/migrations"
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
