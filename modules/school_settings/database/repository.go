package database

import (
	"github.com/elcomware/edupilot/modules/school_settings/app/models"
	"github.com/elcomware/edupilot/modules/school_settings/app/services"
	"github.com/elcomware/edupilot/platform/database"
	"github.com/elcomware/edupilot/platform/workspace/sqlstore"
)

func New(catalog *database.Catalog) services.Repository {
	return sqlstore.New(catalog, models.Key)
}
