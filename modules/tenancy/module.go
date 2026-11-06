// Package tenancy exposes the integration surface of the Schools module.
package tenancy

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/tenancy/app/services"
	records "github.com/elcomware/edupilot/modules/tenancy/database"
	"github.com/elcomware/edupilot/modules/tenancy/database/migrations"

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
