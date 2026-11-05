// Package subscriptions exposes the integration surface of the Subscriptions module.
package subscriptions

import (
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/modules/subscriptions/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/subscriptions/application"
	"github.com/elcomware/edupilot-os/modules/subscriptions/migrations"

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
