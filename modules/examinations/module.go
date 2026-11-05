// Package examinations exposes the integration surface of the Examinations module.
package examinations

import (
	"github.com/elcomware/edupilot-os/modules/examinations/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/examinations/application"
	"github.com/elcomware/edupilot-os/modules/examinations/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "examinations", Group: "academics", Title: catalog.Title{EN: "Examinations", FR: "Examens"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
