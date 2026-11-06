package migrations

import (
	"github.com/elcomware/edupilot/modules/core/migration"
)

func Workspace(db migration.Schema) error { return migration.Workspace(db, "ws_tenancy") }
