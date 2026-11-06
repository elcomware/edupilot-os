// Package messaging exposes the integration surface of the Messaging module.
package messaging

import (
	"github.com/elcomware/edupilot/modules/messaging/app/services"
	records "github.com/elcomware/edupilot/modules/messaging/database"
	"github.com/elcomware/edupilot/modules/messaging/database/migrations"
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
