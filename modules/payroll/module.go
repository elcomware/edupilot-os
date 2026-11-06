// Package payroll exposes the integration surface of the Payroll module.
package payroll

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/payroll/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/payroll/application"
	"github.com/elcomware/edupilot/modules/payroll/migrations"

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
