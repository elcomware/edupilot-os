// Package auth owns school accounts, memberships, roles, grants and sessions.
package auth

import (
	"github.com/elcomware/edupilot/modules/core/contracts/modulekit"
)

func Definition() modulekit.Definition {
	return modulekit.Definition{
		Descriptor: Descriptor(),
	}
}
