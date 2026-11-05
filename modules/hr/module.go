// Package hr exposes the integration surface of the Human resources module.
package hr

import (
	"github.com/elcomware/edupilot-os/modules/hr/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/hr/application"
	"github.com/elcomware/edupilot-os/modules/hr/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "hr", Group: "operations", Title: catalog.Title{EN: "Human resources", FR: "Ressources humaines"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
