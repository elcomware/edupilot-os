// Package modulekit is the contract between module-owned wiring and the composition root.
package modulekit

import (
	"github.com/elcomware/edupilot/modules/core/catalog"
	"github.com/elcomware/edupilot/modules/core/migration"
)

type Definition struct {
	catalog.Descriptor

	WorkspaceSchema func(migration.Schema) error
}
