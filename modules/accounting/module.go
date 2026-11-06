// Package accounting exposes the integration surface of the Accounting module.
package accounting

import (
	"github.com/elcomware/edupilot/modules/accounting/app/services"
	records "github.com/elcomware/edupilot/modules/accounting/database"
	"github.com/elcomware/edupilot/modules/accounting/database/migrations"
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
