// Package accounting exposes the integration surface of the Accounting module.
package accounting

import (
	"github.com/elcomware/edupilot-os/modules/accounting/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/accounting/application"
	"github.com/elcomware/edupilot-os/modules/accounting/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"

	"github.com/elcomware/edupilot-os/platform/database"
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
