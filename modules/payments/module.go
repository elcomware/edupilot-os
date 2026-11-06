// Package payments exposes the integration surface of the Payments module.
package payments

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/payments/app/services"
	records "github.com/elcomware/edupilot/modules/payments/database"
	"github.com/elcomware/edupilot/modules/payments/database/migrations"

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
