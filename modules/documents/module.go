// Package documents exposes the integration surface of the Documents module.
package documents

import (
	"github.com/elcomware/edupilot-os/modules/documents/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/documents/application"
	"github.com/elcomware/edupilot-os/modules/documents/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "documents", Group: "engagement", Title: catalog.Title{EN: "Documents", FR: "Documents"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
