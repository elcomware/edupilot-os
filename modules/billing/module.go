// Package billing exposes the integration surface of the Billing module.
package billing

import (
	"github.com/elcomware/edupilot/modules/billing/app/services"
	records "github.com/elcomware/edupilot/modules/billing/database"
	"github.com/elcomware/edupilot/modules/billing/database/migrations"
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
