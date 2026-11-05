// Package reporting exposes the integration surface of the Reports module.
package reporting

import (
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/modules/reporting/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/reporting/application"
	"github.com/elcomware/edupilot-os/modules/reporting/migrations"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "reporting", Group: "engagement", Title: catalog.Title{EN: "Reports", FR: "Rapports"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
