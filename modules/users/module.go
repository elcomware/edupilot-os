// Package users exposes the integration surface of the User profiles module.
package users

import (
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/modules/users/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/users/application"
	"github.com/elcomware/edupilot-os/modules/users/migrations"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "users", Group: "platform", Title: catalog.Title{EN: "User profiles", FR: "Profils utilisateurs"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
