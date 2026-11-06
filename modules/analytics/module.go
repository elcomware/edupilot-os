// Package analytics exposes the integration surface of the Analytics module.
package analytics

import (
	"github.com/elcomware/edupilot/modules/analytics/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/analytics/application"
	"github.com/elcomware/edupilot/modules/analytics/migrations"
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
