// Package fees exposes the integration surface of the Fee policies module.
package fees

import (
	"github.com/elcomware/edupilot-os/modules/fees/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/fees/application"
	"github.com/elcomware/edupilot-os/modules/fees/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "fees", Group: "finance", Title: catalog.Title{EN: "Fee policies", FR: "Politique tarifaire"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
