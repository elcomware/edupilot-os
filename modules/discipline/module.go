// Package discipline exposes the integration surface of the Discipline and wellbeing module.
package discipline

import (
	"github.com/elcomware/edupilot-os/modules/discipline/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/discipline/application"
	"github.com/elcomware/edupilot-os/modules/discipline/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "discipline", Group: "lifecycle", Title: catalog.Title{EN: "Discipline and wellbeing", FR: "Discipline et bien-être"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
