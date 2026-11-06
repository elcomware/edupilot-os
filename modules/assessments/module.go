// Package assessments exposes the integration surface of the Assessments module.
package assessments

import (
	"github.com/elcomware/edupilot/modules/assessments/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/assessments/application"
	"github.com/elcomware/edupilot/modules/assessments/migrations"
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
