package persistence

import (
	"github.com/elcomware/edupilot/modules/billing/application"
	"github.com/elcomware/edupilot/modules/billing/internal/domain"
	"github.com/elcomware/edupilot/platform/database"
	"github.com/elcomware/edupilot/platform/workspace/sqlstore"
)

func New(catalog *database.Catalog) application.Repository {
	return sqlstore.New(catalog, domain.Key)
}
