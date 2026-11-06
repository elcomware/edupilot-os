// Package curriculum exposes the integration surface of the Curriculum module.
package curriculum

import (
	"github.com/elcomware/edupilot/modules/curriculum/app/services"
	records "github.com/elcomware/edupilot/modules/curriculum/database"
	"github.com/elcomware/edupilot/modules/curriculum/database/migrations"
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
