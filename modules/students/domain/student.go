// Package domain owns the student record and its lifecycle; it does not know
// how a school is authenticated, where a record is stored or how it is served.
package domain

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Status string

const (
	Active   Status = "active"
	Archived Status = "archived"
)

var (
	ErrInvalid  = errors.New("invalid student")
	ErrNotFound = errors.New("student not found")
	ErrConflict = errors.New("student version conflict")
	ErrArchived = errors.New("student is archived")
)

type Student struct {
	id, school, given, family string
	version                   int64
	status                    Status
}

func (s Student) ID() string         { return s.id }
func (s Student) School() string     { return s.school }
func (s Student) GivenName() string  { return s.given }
func (s Student) FamilyName() string { return s.family }
func (s Student) Version() int64     { return s.version }
func (s Student) Status() Status     { return s.status }

// Names retain their spelling, punctuation and internal spaces. Leading and
// trailing spaces are formatting; control characters are never part of a name.
func name(value, field string) (string, error) {
	if !utf8.ValidString(value) || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("%w: %s contains a control character or invalid UTF-8", ErrInvalid, field)
	}
	value = strings.TrimSpace(value)
	if n := utf8.RuneCountInString(value); n < 1 || n > 100 {
		return "", fmt.Errorf("%w: %s must contain 1 to 100 characters", ErrInvalid, field)
	}
	return value, nil
}

func Register(id, school, given, family string) (Student, error) {
	return Restore(id, school, given, family, 1, Active)
}

// Restore validates persisted state before it can participate in a command.
func Restore(id, school, given, family string, version int64, status Status) (Student, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(school) == "" || version < 1 || (status != Active && status != Archived) {
		return Student{}, fmt.Errorf("%w: identity, school, version or status is invalid", ErrInvalid)
	}
	given, err := name(given, "givenName")
	if err != nil {
		return Student{}, err
	}
	family, err = name(family, "familyName")
	if err != nil {
		return Student{}, err
	}
	return Student{id: id, school: school, given: given, family: family, version: version, status: status}, nil
}

func (s Student) change(expected int64) error {
	switch {
	case expected < 1:
		return fmt.Errorf("%w: version must be positive", ErrInvalid)
	case expected != s.version:
		return ErrConflict
	case s.status == Archived:
		return ErrArchived
	}
	return nil
}

func (s Student) Rename(expected int64, given, family string) (Student, error) {
	if err := s.change(expected); err != nil {
		return Student{}, err
	}
	return Restore(s.id, s.school, given, family, s.version+1, Active)
}

func (s Student) Archive(expected int64) (Student, error) {
	if err := s.change(expected); err != nil {
		return Student{}, err
	}
	return Restore(s.id, s.school, s.given, s.family, s.version+1, Archived)
}
