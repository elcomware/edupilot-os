// Package audit exposes the integration surface of the Audit module.
package audit

import (
	"github.com/elcomware/edupilot/modules/audit/app/services"
	records "github.com/elcomware/edupilot/modules/audit/database"
	"github.com/elcomware/edupilot/modules/audit/database/migrations"
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
