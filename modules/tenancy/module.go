// Package tenancy exposes the integration surface of the Schools module.
package tenancy

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/tenancy/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/tenancy/application"
	"github.com/elcomware/edupilot/modules/tenancy/migrations"

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
