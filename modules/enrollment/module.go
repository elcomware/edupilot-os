// Package enrollment exposes the integration surface of the Enrollment and transfers module.
package enrollment

import (
	"github.com/elcomware/edupilot/modules/enrollment/app/services"
	records "github.com/elcomware/edupilot/modules/enrollment/database"
	"github.com/elcomware/edupilot/modules/enrollment/database/migrations"
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
