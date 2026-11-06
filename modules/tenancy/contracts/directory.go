// Package contracts exposes topology lookups for other EduPilot OS modules.
package contracts

import (
	"context"

	"github.com/elcomware/edupilot-os/platform/scope"
)

// Directory verifies ancestry. Permission checks belong to verified request handling.
type Directory interface {
	Resolve(context.Context, scope.Selection) (scope.Chain, error)
}
