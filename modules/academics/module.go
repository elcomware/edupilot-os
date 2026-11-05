// Package academics exposes the integration surface of the Academic planning module.
package academics

import (
	"github.com/elcomware/edupilot-os/modules/academics/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/academics/application"
	"github.com/elcomware/edupilot-os/modules/academics/migrations"
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
