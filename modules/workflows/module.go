// Package workflows exposes the integration surface of the Workflows module.
package workflows

import (
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/modules/workflows/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/workflows/application"
	"github.com/elcomware/edupilot-os/modules/workflows/migrations"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "workflows", Group: "engagement", Title: catalog.Title{EN: "Workflows", FR: "Circuits de validation"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
