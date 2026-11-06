// Package notifications exposes the integration surface of the Notifications module.
package notifications

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/notifications/app/services"
	records "github.com/elcomware/edupilot/modules/notifications/database"
	"github.com/elcomware/edupilot/modules/notifications/database/migrations"

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
