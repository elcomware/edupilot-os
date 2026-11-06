// Package providers is the Students composition point used by the application's
// Goravel service provider. Its adapters share one application service.
package providers

import (
	"net/http"

	studenthttp "github.com/elcomware/edupilot/modules/students/adapters/inbound/http"
	"github.com/elcomware/edupilot/modules/students/adapters/outbound/persistence"
	"github.com/elcomware/edupilot/modules/students/application"
	"github.com/elcomware/edupilot/modules/students/contracts"
	"github.com/elcomware/edupilot/platform/database"
)

type Runtime struct {
	HTTP      http.Handler
	Directory contracts.Directory
}

func New(catalog *database.Catalog) Runtime {
	service := application.New(persistence.New(catalog))
	return Runtime{HTTP: studenthttp.New(service), Directory: service}
}
