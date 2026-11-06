// Package scholarships exposes the integration surface of the Scholarships module.
package scholarships

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/scholarships/app/services"
	records "github.com/elcomware/edupilot/modules/scholarships/database"
	"github.com/elcomware/edupilot/modules/scholarships/database/migrations"

	"github.com/elcomware/edupilot/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: Descriptor(),
		Bind: func(db *database.Catalog) services.Entries {
			return services.Entries{Store: records.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
