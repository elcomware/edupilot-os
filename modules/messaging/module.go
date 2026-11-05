// Package messaging exposes the integration surface of the Messaging module.
package messaging

import (
	"github.com/elcomware/edupilot-os/modules/messaging/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/messaging/application"
	"github.com/elcomware/edupilot-os/modules/messaging/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"

	"github.com/elcomware/edupilot-os/platform/database"
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
