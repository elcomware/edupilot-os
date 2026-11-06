// Package scholarships exposes the integration surface of the Scholarships module.
package scholarships

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/scholarships/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/scholarships/application"
	"github.com/elcomware/edupilot/modules/scholarships/migrations"

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
