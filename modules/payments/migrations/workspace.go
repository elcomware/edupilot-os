package migrations

import (
	"github.com/elcomware/edupilot-os/database/edumigrations"
	"github.com/elcomware/edupilot-os/modules/payments/internal/domain"
	"github.com/elcomware/edupilot-os/platform/workspace/sqlstore"
)

// WorkspaceSchema belongs to this module and runs on shared and dedicated databases.
func WorkspaceSchema() edumigrations.Plan { return sqlstore.Schema(domain.Key) }
