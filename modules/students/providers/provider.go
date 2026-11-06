// Package providers is the Students composition point used by the application's
// Goravel service provider. Its adapters share one application service.
package providers

import (
	"net/http"

	"github.com/elcomware/edupilot/modules/students/app/contracts"
	studenthttp "github.com/elcomware/edupilot/modules/students/app/http/controllers"
	"github.com/elcomware/edupilot/modules/students/app/services"
	records "github.com/elcomware/edupilot/modules/students/database"
	"github.com/elcomware/edupilot/platform/database"
)

type Runtime struct {
	HTTP      http.Handler
	Directory contracts.Directory
}

func New(catalog *database.Catalog) Runtime {
	service := services.New(records.New(catalog))
	return Runtime{HTTP: studenthttp.New(service), Directory: service}
}
