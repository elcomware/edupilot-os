// Package payments exposes the integration surface of the Payments module.
package payments

import (
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/modules/payments/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/payments/application"
	"github.com/elcomware/edupilot-os/modules/payments/migrations"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "payments", Group: "finance", Title: catalog.Title{EN: "Payments", FR: "Paiements"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
