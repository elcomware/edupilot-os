// Package permissions exposes the integration surface of the Permissions module.
package permissions

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/permissions/app/services"
	records "github.com/elcomware/edupilot/modules/permissions/database"
	"github.com/elcomware/edupilot/modules/permissions/database/migrations"

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
