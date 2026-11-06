// Package subscriptions exposes the integration surface of the Subscriptions module.
package subscriptions

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/subscriptions/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/subscriptions/application"
	"github.com/elcomware/edupilot/modules/subscriptions/migrations"

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
