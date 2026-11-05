// Package audit exposes the integration surface of the Audit module.
package audit

import (
	"github.com/elcomware/edupilot-os/modules/audit/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/audit/application"
	"github.com/elcomware/edupilot-os/modules/audit/migrations"
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
