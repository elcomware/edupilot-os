// Package testkit supplies an in-memory application port for Students tests.
// The production provider always constructs the SQL adapter.
package testkit

import (
	"cmp"
	"context"
	"slices"

	"github.com/elcomware/edupilot/modules/students/app/contracts"
	"github.com/elcomware/edupilot/modules/students/app/models"
)

type Memory struct {
	records map[[2]string]models.Student
	Writes  []string
}

func NewMemory() *Memory { return &Memory{records: make(map[[2]string]models.Student)} }

var _ contracts.Repository = (*Memory)(nil)

func (m *Memory) Insert(_ context.Context, s models.Student, actor string) error {
	key := [2]string{s.School(), s.ID()}
	if _, exists := m.records[key]; exists {
		return models.ErrConflict
	}
	m.records[key] = s
	m.Writes = append(m.Writes, actor)
	return nil
}

func (m *Memory) Get(_ context.Context, school, id string) (models.Student, error) {
	s, ok := m.records[[2]string{school, id}]
	if !ok {
		return models.Student{}, models.ErrNotFound
	}
	return s, nil
}

func (m *Memory) Replace(_ context.Context, s models.Student, expected int64, actor string) error {
	key := [2]string{s.School(), s.ID()}
	prev, ok := m.records[key]
	if !ok || prev.Version() != expected || prev.Status() != models.Active {
		return models.ErrConflict
	}
	m.records[key] = s
	m.Writes = append(m.Writes, actor)
	return nil
}

func (m *Memory) ActivePage(_ context.Context, school string, limit, offset int) ([]models.Student, int64, error) {
	var all []models.Student
	for _, s := range m.records {
		if s.School() == school && s.Status() == models.Active {
			all = append(all, s)
		}
	}
	slices.SortFunc(all, func(a, b models.Student) int {
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
