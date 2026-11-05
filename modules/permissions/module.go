// Package permissions exposes the integration surface of the Permissions module.
package permissions

import (
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/modules/permissions/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/permissions/application"
	"github.com/elcomware/edupilot-os/modules/permissions/migrations"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "permissions", Group: "platform", Title: catalog.Title{EN: "Permissions", FR: "Autorisations"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
