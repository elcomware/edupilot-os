package database

import (
	"github.com/elcomware/edupilot/modules/reporting/app/models"
	"github.com/elcomware/edupilot/modules/reporting/app/services"
	"github.com/elcomware/edupilot/platform/database"
	"github.com/elcomware/edupilot/platform/workspace/sqlstore"
)

func New(catalog *database.Catalog) services.Repository {
	return sqlstore.New(catalog, models.Key)
}
