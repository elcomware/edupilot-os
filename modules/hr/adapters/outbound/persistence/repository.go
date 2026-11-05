package persistence

import (
	"github.com/elcomware/edupilot-os/modules/hr/application"
	"github.com/elcomware/edupilot-os/modules/hr/internal/domain"
	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/elcomware/edupilot-os/platform/workspace/sqlstore"
)

func New(catalog *database.Catalog) application.Repository {
	return sqlstore.New(catalog, domain.Key)
}
