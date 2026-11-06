// Package tenancy exposes the integration surface of the Schools module.
package tenancy

import (
	"github.com/elcomware/edupilot/modules/core/contracts/modulekit"
	"github.com/elcomware/edupilot/modules/tenancy/database/migrations"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor:      Descriptor(),
		WorkspaceSchema: migrations.Workspace,
	}
}
