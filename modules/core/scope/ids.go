// Package scope defines placement-independent identity and ownership. Callers
// must verify effective grants separately from parsing a resource identifier.
package scope

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

type (
	Tenant       struct{}
	Organization struct{}
	School       struct{}
	Campus       struct{}
	Location     struct{}
	Team         struct{}
	Curriculum   struct{}
	Subject      struct{}
	Membership   struct{}
	Pool         struct{}
	Session      struct{}
)
type kind interface {
	Tenant | Organization | School | Campus | Location | Team | Curriculum | Subject | Membership | Pool | Session
}

// ID is comparable. The kind field also prevents explicit cross-kind conversions.
type ID[K kind] struct {
	_     [0]K
	value uuid.UUID
}
type (
	TenantID       = ID[Tenant]
	OrganizationID = ID[Organization]
	SchoolID       = ID[School]
	CampusID       = ID[Campus]
	LocationID     = ID[Location]
	TeamID         = ID[Team]
	CurriculumID   = ID[Curriculum]
	SubjectID      = ID[Subject]
	MembershipID   = ID[Membership]
	PoolID         = ID[Pool]
	SessionID      = ID[Session]
)

var ErrInvalidID = errors.New("scope: expected a canonical random UUID")

func NewID[K kind]() (ID[K], error) {
	value, err := uuid.NewRandom()
	if err != nil {
		return ID[K]{}, err
	}
	return ID[K]{value: value}, nil
}
func Parse[K kind](raw string) (ID[K], error) {
	value, err := uuid.Parse(raw)
	if err != nil || value.Version() != 4 || value.Variant() != uuid.RFC4122 || value.String() != raw {
		return ID[K]{}, ErrInvalidID
	}
	return ID[K]{value: value}, nil
}
func (id ID[K]) IsZero() bool   { return id.value == uuid.Nil }
func (id ID[K]) String() string { return id.value.String() }
func (id ID[K]) MarshalText() ([]byte, error) {
	if id.IsZero() {
		return nil, ErrInvalidID
	}
	return []byte(id.String()), nil
}
func (id *ID[K]) UnmarshalText(data []byte) error {
	parsed, err := Parse[K](string(data))
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

// UnmarshalJSON rejects a null ID even when decoding into an existing value.
func (id *ID[K]) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return ErrInvalidID
	}
	return id.UnmarshalText([]byte(value))
}
