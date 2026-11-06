// Package admissions exposes the integration surface of the Admissions module.
package admissions

import (
	"github.com/elcomware/edupilot/modules/admissions/app/services"
	records "github.com/elcomware/edupilot/modules/admissions/database"
	"github.com/elcomware/edupilot/modules/admissions/database/migrations"
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
