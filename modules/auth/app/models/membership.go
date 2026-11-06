// Package models defines the identities and relationships owned by Auth.
package models

type Membership struct {
	School string `json:"school"`
	UserID string `json:"userId"`
	Role   Role   `json:"role"`
}
