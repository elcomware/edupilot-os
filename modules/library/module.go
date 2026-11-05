// Package library exposes the integration surface of the Library module.
package library

import (
	"github.com/elcomware/edupilot-os/modules/library/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/library/application"
	"github.com/elcomware/edupilot-os/modules/library/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "library", Group: "operations", Title: catalog.Title{EN: "Library", FR: "Bibliothèque"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
