// Package modulekit is the contract between module-owned wiring and the composition root.
package modulekit

import (
	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/platform/catalog"
	"github.com/elcomware/edupilot/platform/database"
	"github.com/elcomware/edupilot/platform/workspace"
)

type Definition struct {
	catalog.Descriptor
	Bind      func(*database.Catalog) workspace.Service
	Migration func() edumigrations.Plan
}
