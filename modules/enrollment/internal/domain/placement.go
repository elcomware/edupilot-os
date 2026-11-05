package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrInvalid       = errors.New("invalid placement")
	ErrYearOccupied  = errors.New("student already has a placement for this academic year")
	ErrWithdrawn     = errors.New("placement has been withdrawn")
	ErrStale         = errors.New("placement version conflict")
	ErrPlacementMiss = errors.New("placement not found")
	ErrStudentMiss   = errors.New("student not found")
	ErrStudentClosed = errors.New("student is archived")
)

const (
	ActivePlacement    = "active"
	WithdrawnPlacement = "withdrawn"
)

type Placement struct {
	ID, StudentID, AcademicYear, Grade, Course, Status string
	Version                                            int64
}

var yearPattern = regexp.MustCompile(`^[1-9][0-9]{3}-[1-9][0-9]{3}$`)

func NewPlacement(id, studentID, academicYear, grade, course string) (Placement, error) {
	if id == "" || studentID == "" || len(studentID) > 64 || strings.IndexFunc(studentID, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
		return Placement{}, fmt.Errorf("%w: studentId must be a valid student ID", ErrInvalid)
	}
	if !yearPattern.MatchString(academicYear) {
		return Placement{}, fmt.Errorf("%w: academicYear must use YYYY-YYYY", ErrInvalid)
	}
	start, _ := strconv.Atoi(academicYear[:4])
	end, _ := strconv.Atoi(academicYear[5:])
	if end != start+1 {
		return Placement{}, fmt.Errorf("%w: academicYear must span consecutive years", ErrInvalid)
	}
	grade, err := label(grade, "grade", 1, 48)
	if err != nil {
		return Placement{}, err
	}
	course, err = label(course, "course", 0, 120)
	if err != nil {
		return Placement{}, err
	}
	return Placement{ID: id, StudentID: studentID, AcademicYear: academicYear, Grade: grade, Course: course, Status: ActivePlacement, Version: 1}, nil
}

func label(value, field string, min, max int) (string, error) {
	if !utf8.ValidString(value) || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("%w: invalid %s", ErrInvalid, field)
	}
	value = strings.TrimSpace(value)
	count := utf8.RuneCountInString(value)
	if count < min || count > max {
		return "", fmt.Errorf("%w: %s must contain %d to %d characters", ErrInvalid, field, min, max)
	}
	return value, nil
}

func (p Placement) Retry(proposed Placement) error {
	if p.Status == WithdrawnPlacement {
		return ErrWithdrawn
	}
	if p.Grade != proposed.Grade || p.Course != proposed.Course {
		return ErrYearOccupied
	}
	return nil
}

func (p Placement) Withdraw(expected int64) (Placement, error) {
	if expected < 1 {
		return Placement{}, fmt.Errorf("%w: version must be positive", ErrInvalid)
	}
	if p.Version != expected {
		return Placement{}, ErrStale
	}
	if p.Status != ActivePlacement {
		return Placement{}, ErrWithdrawn
	}
	p.Status = WithdrawnPlacement
	p.Version++
	return p, nil
}
