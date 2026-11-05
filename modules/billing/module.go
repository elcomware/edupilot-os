// Package billing exposes the integration surface of the Billing module.
package billing

import (
	"github.com/elcomware/edupilot-os/modules/billing/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/billing/application"
	"github.com/elcomware/edupilot-os/modules/billing/migrations"
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
