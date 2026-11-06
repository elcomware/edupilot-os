// Package school_settings exposes the integration surface of the School settings module.
package school_settings

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/school_settings/app/services"
	records "github.com/elcomware/edupilot/modules/school_settings/database"
	"github.com/elcomware/edupilot/modules/school_settings/database/migrations"

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
