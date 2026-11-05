// Package procurement exposes the integration surface of the Procurement module.
package procurement

import (
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/modules/procurement/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/procurement/application"
	"github.com/elcomware/edupilot-os/modules/procurement/migrations"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "procurement", Group: "finance", Title: catalog.Title{EN: "Procurement", FR: "Achats"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
