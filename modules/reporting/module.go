// Package reporting exposes the integration surface of the Reports module.
package reporting

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/reporting/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/reporting/application"
	"github.com/elcomware/edupilot/modules/reporting/migrations"

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
