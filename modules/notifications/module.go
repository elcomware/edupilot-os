// Package notifications exposes the integration surface of the Notifications module.
package notifications

import (
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/modules/notifications/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/notifications/application"
	"github.com/elcomware/edupilot-os/modules/notifications/migrations"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "notifications", Group: "engagement", Title: catalog.Title{EN: "Notifications", FR: "Notifications"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
