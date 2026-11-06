// Package providers wires Enrollment's placement capability for the API root.
// Students is accessible solely through its public Directory contract.
package providers

import (
	"net/http"

	enrollmenthttp "github.com/elcomware/edupilot/modules/enrollment/app/http/controllers"
	"github.com/elcomware/edupilot/modules/enrollment/app/services"
	records "github.com/elcomware/edupilot/modules/enrollment/database"
	"github.com/elcomware/edupilot/modules/enrollment/database/migrations"
	"github.com/elcomware/edupilot/modules/students/app/contracts"
	"github.com/elcomware/edupilot/platform/database"
)

var SchemaUp = migrations.PlacementsUp
var SchemaDown = migrations.PlacementsDown

func New(catalog *database.Catalog, students contracts.Directory) http.Handler {
	return enrollmenthttp.New(services.Placements{
		Store: records.PlacementRepository{Catalog: catalog}, Students: students,
	})
}
