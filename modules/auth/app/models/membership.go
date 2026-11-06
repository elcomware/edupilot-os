// Package models defines the identities and relationships owned by Auth.
package models

import "github.com/elcomware/edupilot/platform/access"

type Membership struct {
	School string `json:"school"`
	UserID string `json:"userId"`
	Role   Role   `json:"role"`
}

func (m Membership) Principal() (access.Principal, error) {
	grants, err := m.Role.Permissions()
	if err != nil {
		return access.Principal{}, err
	}
	return access.Principal{ID: m.UserID, School: m.School, Permissions: grants}, nil
}
