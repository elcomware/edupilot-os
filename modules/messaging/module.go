// Package messaging exposes the integration surface of the Messaging module.
package messaging

import (
	"github.com/elcomware/edupilot/modules/messaging/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/messaging/application"
	"github.com/elcomware/edupilot/modules/messaging/migrations"
	"github.com/elcomware/edupilot/modules/modulekit"

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
