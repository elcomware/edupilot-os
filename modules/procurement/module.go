// Package procurement exposes the integration surface of the Procurement module.
package procurement

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/procurement/app/services"
	records "github.com/elcomware/edupilot/modules/procurement/database"
	"github.com/elcomware/edupilot/modules/procurement/database/migrations"

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
