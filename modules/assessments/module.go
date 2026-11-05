// Package assessments exposes the integration surface of the Assessments module.
package assessments

import (
	"github.com/elcomware/edupilot-os/modules/assessments/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/assessments/application"
	"github.com/elcomware/edupilot-os/modules/assessments/migrations"
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
