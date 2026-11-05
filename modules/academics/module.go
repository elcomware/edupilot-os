// Package academics exposes the integration surface of the Academic planning module.
package academics

import (
	"github.com/elcomware/edupilot-os/modules/academics/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/academics/application"
	"github.com/elcomware/edupilot-os/modules/academics/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "academics", Group: "academics", Title: catalog.Title{EN: "Academic planning", FR: "Organisation pédagogique"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
