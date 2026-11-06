// Package payments exposes the integration surface of the Payments module.
package payments

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/payments/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/payments/application"
	"github.com/elcomware/edupilot/modules/payments/migrations"

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
