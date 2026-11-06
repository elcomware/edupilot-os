// Package transcripts exposes the integration surface of the Transcripts module.
package transcripts

import (
	"github.com/elcomware/edupilot/modules/modulekit"
	"github.com/elcomware/edupilot/modules/transcripts/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/transcripts/application"
	"github.com/elcomware/edupilot/modules/transcripts/migrations"

	"github.com/elcomware/edupilot/platform/database"
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
