// Package workspace provides the school-scoped work list that module pages share.
// A work item is a planning note; specialized business commands have separate ports.
package workspace

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrInvalid  = errors.New("invalid workspace entry")
	ErrMissing  = errors.New("entry not found")
	ErrConflict = errors.New("entry version changed")
)

type Entry struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Version int64  `json:"version"`
}
type Page struct {
	Items []Entry `json:"items"`
	Total int     `json:"total"`
}

// Repository methods must isolate every read and write by school. Update uses
// compare-and-swap; a successful mutation and its event commit together.
type Repository interface {
	Get(context.Context, string, string) (Entry, error)
	List(context.Context, string, int, int) (Page, error)
	Create(context.Context, string, string) (Entry, error)
	Update(context.Context, string, Entry) (Entry, error)
}
type Service struct{ Store Repository }

func (s Service) List(ctx context.Context, school string, limit, offset int) (Page, error) {
	if strings.TrimSpace(school) == "" || limit < 1 || limit > 100 || offset < 0 {
		return Page{}, ErrInvalid
	}
	return s.Store.List(ctx, school, limit, offset)
}
func validTitle(title string) bool { return title != "" && utf8.RuneCountInString(title) <= 160 }
func (s Service) Create(ctx context.Context, school, title string) (Entry, error) {
	title = strings.TrimSpace(title)
	if strings.TrimSpace(school) == "" || !validTitle(title) {
		return Entry{}, ErrInvalid
	}
	return s.Store.Create(ctx, school, title)
}
func (s Service) Update(ctx context.Context, school string, item Entry) (Entry, error) {
	item.Title = strings.TrimSpace(item.Title)
	if strings.TrimSpace(school) == "" || item.ID == "" || !validTitle(item.Title) || (item.Status != "open" && item.Status != "closed") || item.Version < 1 {
		return Entry{}, ErrInvalid
	}
	return s.Store.Update(ctx, school, item)
}

func (s Service) Get(ctx context.Context, school, id string) (Entry, error) {
	if strings.TrimSpace(school) == "" || strings.TrimSpace(id) == "" {
		return Entry{}, ErrInvalid
	}
	return s.Store.Get(ctx, school, id)
}
