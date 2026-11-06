// Package inventory exposes the integration surface of the Inventory module.
package inventory

import (
	"github.com/elcomware/edupilot/modules/inventory/app/services"
	records "github.com/elcomware/edupilot/modules/inventory/database"
	"github.com/elcomware/edupilot/modules/inventory/database/migrations"
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
