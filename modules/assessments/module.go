// Package assessments exposes the integration surface of the Assessments module.
package assessments

import (
	"github.com/elcomware/edupilot/modules/assessments/app/services"
	records "github.com/elcomware/edupilot/modules/assessments/database"
	"github.com/elcomware/edupilot/modules/assessments/database/migrations"
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
