package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/elcomware/edupilot/modules/students/application/ports"
	"github.com/elcomware/edupilot/modules/students/contracts"
	"github.com/elcomware/edupilot/modules/students/domain"
)

const DefaultLimit = 25
const MaximumLimit = 100

type Service struct{ repo ports.Repository }

func New(repo ports.Repository) *Service { return &Service{repo: repo} }

var _ contracts.Directory = (*Service)(nil)

func view(s domain.Student) contracts.Student {
	return contracts.Student{ID: s.ID(), GivenName: s.GivenName(), FamilyName: s.FamilyName(), Version: s.Version(), Status: string(s.Status())}
}

func schoolRequired(school string) error {
	if strings.TrimSpace(school) == "" {
		return fmt.Errorf("%w: a school is required", domain.ErrInvalid)
	}
	return nil
}

func actorRequired(actor string) error {
	if strings.TrimSpace(actor) == "" {
		return fmt.Errorf("%w: an actor is required", domain.ErrInvalid)
	}
	return nil
}

func (s *Service) Register(ctx context.Context, school, actor, given, family string) (contracts.Student, error) {
	if err := actorRequired(actor); err != nil {
		return contracts.Student{}, err
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return contracts.Student{}, fmt.Errorf("generate student id: %w", err)
	}
	student, err := domain.Register(hex.EncodeToString(id[:]), school, given, family)
	if err != nil {
		return contracts.Student{}, err
	}
	if err = s.repo.Insert(ctx, student, actor); err != nil {
		return contracts.Student{}, err
	}
	return view(student), nil
}

func (s *Service) Lookup(ctx context.Context, school, id string) (contracts.Student, error) {
	if err := schoolRequired(school); err != nil {
		return contracts.Student{}, err
	}
	student, err := s.repo.Get(ctx, school, id)
	if errors.Is(err, domain.ErrNotFound) {
		return contracts.Student{}, contracts.ErrNotFound
	}
	if err != nil {
		return contracts.Student{}, err
	}
	return view(student), nil
}

func (s *Service) List(ctx context.Context, school string, limit, offset int) (contracts.Page, error) {
	if err := schoolRequired(school); err != nil {
		return contracts.Page{}, err
	}
	if limit < 1 || limit > MaximumLimit || offset < 0 {
		return contracts.Page{}, fmt.Errorf("%w: limit must be 1 to %d and offset nonnegative", domain.ErrInvalid, MaximumLimit)
	}
	items, total, err := s.repo.ActivePage(ctx, school, limit, offset)
	if err != nil {
		return contracts.Page{}, err
	}
	page := contracts.Page{Items: make([]contracts.Student, 0, len(items)), Total: total}
	for _, student := range items {
		page.Items = append(page.Items, view(student))
	}
	return page, nil
}

func (s *Service) Update(ctx context.Context, school, actor, id string, expected int64, given, family string) (contracts.Student, error) {
	return s.mutate(ctx, school, actor, id, expected, func(current domain.Student) (domain.Student, error) { return current.Rename(expected, given, family) })
}

func (s *Service) Archive(ctx context.Context, school, actor, id string, expected int64) error {
	_, err := s.mutate(ctx, school, actor, id, expected, func(current domain.Student) (domain.Student, error) { return current.Archive(expected) })
	return err
}

func (s *Service) mutate(ctx context.Context, school, actor, id string, expected int64, change func(domain.Student) (domain.Student, error)) (contracts.Student, error) {
	if err := schoolRequired(school); err != nil {
		return contracts.Student{}, err
	}
	if err := actorRequired(actor); err != nil {
		return contracts.Student{}, err
	}
	if expected < 1 {
		return contracts.Student{}, fmt.Errorf("%w: version must be positive", domain.ErrInvalid)
	}
	current, err := s.repo.Get(ctx, school, id)
	if err != nil {
		return contracts.Student{}, err
	}
	next, err := change(current)
	if err != nil {
		return contracts.Student{}, err
	}
	if err = s.repo.Replace(ctx, next, expected, actor); err != nil {
		return contracts.Student{}, err
	}
	return view(next), nil
}
