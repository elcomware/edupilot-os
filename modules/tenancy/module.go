// Package tenancy exposes the integration surface of the Schools module.
package tenancy

import (
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/modules/tenancy/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/tenancy/application"
	"github.com/elcomware/edupilot-os/modules/tenancy/migrations"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "tenancy", Group: "platform", Title: catalog.Title{EN: "Schools", FR: "Établissements"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
