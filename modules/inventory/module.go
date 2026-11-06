// Package inventory exposes the integration surface of the Inventory module.
package inventory

import (
	"github.com/elcomware/edupilot/modules/inventory/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/inventory/application"
	"github.com/elcomware/edupilot/modules/inventory/migrations"
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
