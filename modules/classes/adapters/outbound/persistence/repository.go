package persistence

import (
	"github.com/elcomware/edupilot/modules/classes/application"
	"github.com/elcomware/edupilot/modules/classes/internal/domain"
	"github.com/elcomware/edupilot/platform/database"
	"github.com/elcomware/edupilot/platform/workspace/sqlstore"
)

func New(catalog *database.Catalog) application.Repository {
	return sqlstore.New(catalog, domain.Key)
}
