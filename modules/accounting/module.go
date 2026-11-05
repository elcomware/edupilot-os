// Package accounting exposes the integration surface of the Accounting module.
package accounting

import (
	"github.com/elcomware/edupilot-os/modules/accounting/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/accounting/application"
	"github.com/elcomware/edupilot-os/modules/accounting/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "accounting", Group: "finance", Title: catalog.Title{EN: "Accounting", FR: "Comptabilité"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
