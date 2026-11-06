// Package testkit supplies an in-memory application port for Students tests.
// The production provider always constructs the SQL adapter.
package testkit

import (
	"cmp"
	"context"
	"slices"

	"github.com/elcomware/edupilot/modules/students/application/ports"
	"github.com/elcomware/edupilot/modules/students/domain"
)

type Memory struct {
	records map[[2]string]domain.Student
	Writes  []string
}

func NewMemory() *Memory { return &Memory{records: make(map[[2]string]domain.Student)} }

var _ ports.Repository = (*Memory)(nil)

func (m *Memory) Insert(_ context.Context, s domain.Student, actor string) error {
	key := [2]string{s.School(), s.ID()}
	if _, exists := m.records[key]; exists {
		return domain.ErrConflict
	}
	m.records[key] = s
	m.Writes = append(m.Writes, actor)
	return nil
}

func (m *Memory) Get(_ context.Context, school, id string) (domain.Student, error) {
	s, ok := m.records[[2]string{school, id}]
	if !ok {
		return domain.Student{}, domain.ErrNotFound
	}
	return s, nil
}

func (m *Memory) Replace(_ context.Context, s domain.Student, expected int64, actor string) error {
	key := [2]string{s.School(), s.ID()}
	prev, ok := m.records[key]
	if !ok || prev.Version() != expected || prev.Status() != domain.Active {
		return domain.ErrConflict
	}
	m.records[key] = s
	m.Writes = append(m.Writes, actor)
	return nil
}

func (m *Memory) ActivePage(_ context.Context, school string, limit, offset int) ([]domain.Student, int64, error) {
	var all []domain.Student
	for _, s := range m.records {
		if s.School() == school && s.Status() == domain.Active {
			all = append(all, s)
		}
	}
	slices.SortFunc(all, func(a, b domain.Student) int {
		if v := cmp.Compare(a.FamilyName(), b.FamilyName()); v != 0 {
			return v
		}
		if v := cmp.Compare(a.GivenName(), b.GivenName()); v != 0 {
			return v
		}
		return cmp.Compare(a.ID(), b.ID())
	})
	total := int64(len(all))
	if offset >= len(all) {
		return nil, total, nil
	}
	return all[offset : offset+min(limit, len(all)-offset)], total, nil
}
