// Package contracts is the only Students API other business modules may import.
package contracts

import (
	"context"
	"errors"
)

// ErrNotFound is returned for unknown records, including records in another school.
var ErrNotFound = errors.New("student not found")

type Student struct {
	ID         string `json:"id"`
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
	Version    int64  `json:"version"`
	Status     string `json:"status"`
}

type Page struct {
	Items []Student `json:"items"`
	Total int64     `json:"total"`
}

// Directory allows Enrollment and other modules to read the authoritative record.
// The caller supplies its already verified school scope. Archived records are
// visible by ID; callers that require an active student must check Status.
type Directory interface {
	Lookup(ctx context.Context, school, id string) (Student, error)
}
