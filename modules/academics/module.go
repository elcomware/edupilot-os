// Package academics exposes the integration surface of the Academic planning module.
package academics

import (
	"github.com/elcomware/edupilot/modules/academics/app/services"
	records "github.com/elcomware/edupilot/modules/academics/database"
	"github.com/elcomware/edupilot/modules/academics/database/migrations"
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
