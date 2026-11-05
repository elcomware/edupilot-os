// Package school_settings exposes the integration surface of the School settings module.
package school_settings

import (
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/modules/school_settings/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/school_settings/application"
	"github.com/elcomware/edupilot-os/modules/school_settings/migrations"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "school_settings", Group: "platform", Title: catalog.Title{EN: "School settings", FR: "Paramètres scolaires"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
