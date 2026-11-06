// Package examinations exposes the integration surface of the Examinations module.
package examinations

import (
	"github.com/elcomware/edupilot/modules/examinations/app/services"
	records "github.com/elcomware/edupilot/modules/examinations/database"
	"github.com/elcomware/edupilot/modules/examinations/database/migrations"
	"github.com/elcomware/edupilot/modules/modulekit"

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
