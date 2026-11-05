package application

import "github.com/elcomware/edupilot-os/platform/workspace"

// Entries is the public use case for this module's school work list.
type Entries = workspace.Service

// Repository is the persistence port accepted by the application.
type Repository interface{ workspace.Repository }
