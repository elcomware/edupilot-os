package providers

import (
	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/modules/auth/database/migrations"
)

func AccountsMigration() edumigrations.Plan {
	return edumigrations.Plan{ID: "20260101000003_school_accounts", ControlOnly: true, Up: migrations.SchemaUp, Down: migrations.SchemaDown}
}

func FoundationMigration() edumigrations.Plan {
	return edumigrations.Plan{ID: "20261106010002_identity", ControlOnly: true, ForwardOnly: true, Up: migrations.FoundationDDL()}
}
