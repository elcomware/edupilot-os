// Package scholarships exposes the integration surface of the Scholarships module.
package scholarships

import (
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/modules/scholarships/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/scholarships/application"
	"github.com/elcomware/edupilot-os/modules/scholarships/migrations"

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
