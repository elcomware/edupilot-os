// Package permissions exposes the integration surface of the Permissions module.
package permissions

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/permissions/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/permissions/application"
	"github.com/elcomware/edupilot/modules/permissions/migrations"

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
