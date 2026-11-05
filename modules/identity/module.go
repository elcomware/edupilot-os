// Package identity exposes the integration surface of the Identity module.
package identity

import (
	"github.com/elcomware/edupilot-os/modules/identity/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/identity/application"
	"github.com/elcomware/edupilot-os/modules/identity/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "identity", Group: "platform", Title: catalog.Title{EN: "Identity", FR: "Identité"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
