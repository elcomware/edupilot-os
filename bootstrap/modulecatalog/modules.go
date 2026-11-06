// Package modulecatalog composes the installed business modules.
package modulecatalog

import (
	"github.com/elcomware/edupilot/modules/auth"
	"github.com/elcomware/edupilot/modules/core/contracts/modulekit"
	"github.com/elcomware/edupilot/modules/tenancy"
)

func Definitions() []modulekit.Definition {
	return []modulekit.Definition{tenancy.Definition(), auth.Definition()}
}
