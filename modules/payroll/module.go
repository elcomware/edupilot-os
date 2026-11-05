// Package payroll exposes the integration surface of the Payroll module.
package payroll

import (
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/modules/payroll/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/payroll/application"
	"github.com/elcomware/edupilot-os/modules/payroll/migrations"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "payroll", Group: "operations", Title: catalog.Title{EN: "Payroll", FR: "Paie"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
