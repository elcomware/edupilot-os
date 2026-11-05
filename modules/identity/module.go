// Package identity exposes the integration surface of the Identity module.
package identity

import (
	"github.com/elcomware/edupilot-os/modules/identity/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/identity/application"
	"github.com/elcomware/edupilot-os/modules/identity/migrations"
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
