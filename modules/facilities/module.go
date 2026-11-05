// Package facilities exposes the integration surface of the Facilities and assets module.
package facilities

import (
	"github.com/elcomware/edupilot-os/modules/facilities/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/facilities/application"
	"github.com/elcomware/edupilot-os/modules/facilities/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "facilities", Group: "operations", Title: catalog.Title{EN: "Facilities and assets", FR: "Locaux et équipements"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
