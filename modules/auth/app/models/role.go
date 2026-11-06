package models

import "github.com/elcomware/edupilot/modules/core/access"

// Role identifies one of the fixed school membership roles.
type Role string

const (
	RoleAdmin     = "admin"
	RoleRegistrar = "registrar"
	RoleViewer    = "viewer"
)

func (r Role) DefaultPermissions() ([]string, error) {
	var grants []string
	switch r {
	case RoleAdmin:
		grants = []string{access.ModulesRead, access.WorkspaceRead, access.WorkspaceWrite, access.ManageMembers}
	case RoleRegistrar:
		grants = []string{access.ModulesRead, access.WorkspaceRead, access.WorkspaceWrite}
	case RoleViewer:
		grants = []string{access.ModulesRead, access.WorkspaceRead}
	default:
		return nil, access.ErrForbidden
	}
	return grants, nil
}
