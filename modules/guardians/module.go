// Package guardians exposes the integration surface of the Guardians module.
package guardians

import (
	"github.com/elcomware/edupilot-os/modules/guardians/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/guardians/application"
	"github.com/elcomware/edupilot-os/modules/guardians/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "guardians", Group: "lifecycle", Title: catalog.Title{EN: "Guardians", FR: "Responsables légaux"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
