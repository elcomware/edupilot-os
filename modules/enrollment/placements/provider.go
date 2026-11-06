// Package placements wires Enrollment's placement capability for the API root.
// Students is accessible solely through its public Directory contract.
package placements

import (
	"net/http"

	enrollmenthttp "github.com/elcomware/edupilot/modules/enrollment/adapters/inbound/http"
	"github.com/elcomware/edupilot/modules/enrollment/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/enrollment/application"
	"github.com/elcomware/edupilot/modules/enrollment/migrations"
	"github.com/elcomware/edupilot/modules/students/contracts"
	"github.com/elcomware/edupilot/platform/database"
)

var SchemaUp = migrations.PlacementsUp
var SchemaDown = migrations.PlacementsDown

func New(catalog *database.Catalog, students contracts.Directory) http.Handler {
	return enrollmenthttp.New(application.Placements{
		Store: persistence.PlacementRepository{Catalog: catalog}, Students: students,
	})
}
