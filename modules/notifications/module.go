// Package notifications exposes the integration surface of the Notifications module.
package notifications

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/notifications/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/notifications/application"
	"github.com/elcomware/edupilot/modules/notifications/migrations"

	"github.com/elcomware/edupilot/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: Descriptor(),
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
