// Package accounting exposes the integration surface of the Accounting module.
package accounting

import (
	"github.com/elcomware/edupilot/modules/accounting/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/accounting/application"
	"github.com/elcomware/edupilot/modules/accounting/migrations"
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
