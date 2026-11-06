// Package hr exposes the integration surface of the Human resources module.
package hr

import (
	"github.com/elcomware/edupilot/modules/hr/app/services"
	records "github.com/elcomware/edupilot/modules/hr/database"
	"github.com/elcomware/edupilot/modules/hr/database/migrations"
	"github.com/elcomware/edupilot/modules/modulekit"

	"github.com/elcomware/edupilot/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: Descriptor(),
		Bind: func(db *database.Catalog) services.Entries {
			return services.Entries{Store: records.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
