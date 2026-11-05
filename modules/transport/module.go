// Package transport exposes the integration surface of the Transport module.
package transport

import (
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/modules/transport/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/transport/application"
	"github.com/elcomware/edupilot-os/modules/transport/migrations"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "transport", Group: "operations", Title: catalog.Title{EN: "Transport", FR: "Transport"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
