package models

// User is the account view safe to return to its owner. A membership is keyed
// by both school and user ID. Password verifiers never enter this view.
type User struct {
	ExpiresAt   int64      `json:"-"`
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	Membership  Membership `json:"membership"`
	Permissions []string   `json:"permissions"`
}
