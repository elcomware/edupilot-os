package models

import "github.com/elcomware/edupilot/platform/access"

// Role identifies one of the fixed school membership roles.
type Role string

const (
	RoleAdmin     = "admin"
	RoleRegistrar = "registrar"
	RoleViewer    = "viewer"
)

func (r Role) Permissions() ([]string, error) {
	var grants []string
	switch r {
	case RoleAdmin:
		grants = []string{access.ModulesRead, access.StudentsRead, access.StudentsWrite, access.WorkspaceRead, access.WorkspaceWrite, access.DocumentsRead, access.DocumentsWrite, access.ManageMembers}
	case RoleRegistrar:
		grants = []string{access.ModulesRead, access.StudentsRead, access.StudentsWrite, access.WorkspaceRead, access.WorkspaceWrite, access.DocumentsRead, access.DocumentsWrite}
	case RoleViewer:
		grants = []string{access.ModulesRead, access.StudentsRead, access.WorkspaceRead, access.DocumentsRead}
	default:
		return nil, access.ErrForbidden
	}
	return grants, nil
}
