// Package assessments exposes the integration surface of the Assessments module.
package assessments

import (
	"github.com/elcomware/edupilot-os/modules/assessments/adapters/outbound/persistence"
	"github.com/elcomware/edupilot-os/modules/assessments/application"
	"github.com/elcomware/edupilot-os/modules/assessments/migrations"
	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/platform/catalog"
	"github.com/elcomware/edupilot-os/platform/database"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: catalog.Descriptor{Key: "assessments", Group: "academics", Title: catalog.Title{EN: "Assessments", FR: "Évaluations"}, Capabilities: []string{"list", "create", "update"}},
		Bind: func(db *database.Catalog) application.Entries {
			return application.Entries{Store: persistence.New(db)}
		},
		Migration: migrations.WorkspaceSchema,
	}
}
