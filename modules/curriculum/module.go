// Package curriculum exposes the integration surface of the Curriculum module.
package curriculum

import (
	"github.com/elcomware/edupilot-os/modules/curriculum/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/curriculum/application"
	"github.com/elcomware/edupilot-os/modules/curriculum/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "curriculum", Group: "academics", Title: catalog.Title{EN: "Curriculum", FR: "Programmes"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
