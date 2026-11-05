// Package portal exposes the integration surface of the Family and student portal module.
package portal

import (
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/modules/portal/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/portal/application"
	"github.com/elcomware/edupilot-os/modules/portal/migrations"

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
