// Package analytics exposes the integration surface of the Analytics module.
package analytics

import (
	"github.com/elcomware/edupilot-os/modules/analytics/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/analytics/application"
	"github.com/elcomware/edupilot-os/modules/analytics/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "analytics", Group: "engagement", Title: catalog.Title{EN: "Analytics", FR: "Analyses"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
