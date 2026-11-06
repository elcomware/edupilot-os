package migrations

import (
	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/modules/assessments/internal/domain"
	"github.com/elcomware/edupilot/platform/workspace/sqlstore"
)

// WorkspaceSchema belongs to this module and runs on shared and dedicated databases.
func WorkspaceSchema() edumigrations.Plan { return sqlstore.Schema(domain.Key) }
