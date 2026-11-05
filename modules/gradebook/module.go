// Package gradebook exposes the integration surface of the Gradebook module.
package gradebook

import (
	"github.com/elcomware/edupilot-os/modules/gradebook/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/gradebook/application"
	"github.com/elcomware/edupilot-os/modules/gradebook/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "gradebook", Group: "academics", Title: catalog.Title{EN: "Gradebook", FR: "Carnet de notes"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
