package services

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/elcomware/edupilot/modules/auth/app/models"
	"github.com/elcomware/edupilot/modules/core/access"
	framework "github.com/goravel/framework/contracts/auth"
)

// Guard exposes the same revocable school session through Goravel Auth(ctx).
type Guard struct {
	Request *http.Request
	Resolve func() (*Service, error)
}

var _ framework.GuardDriver = (*Guard)(nil)

type Credentials struct{ School, Email, Password string }

// Bearer accepts exactly one Authorization value on HTTP and gRPC.
func Bearer(values []string) (string, error) {
	if len(values) != 1 {
		return "", access.ErrUnauthenticated
	}
	fields := strings.Fields(values[0])
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return "", access.ErrUnauthenticated
	}
	return fields[1], nil
}
func (g *Guard) current(token string) (models.User, error) {
	s, err := g.Resolve()
	if err != nil {
		return models.User{}, err
	}
	return s.scopedUser(g.Request.Context(), token, g.Request.Header.Values("X-School-ID"))
}

func (s *Service) scopedUser(ctx context.Context, token string, schools []string) (models.User, error) {
	user, err := s.CurrentUser(ctx, token)
	if err != nil {
		return models.User{}, err
	}
	if len(schools) > 0 && (len(schools) != 1 || schools[0] != user.Membership.School) {
		return models.User{}, access.ErrForbidden
	}
	return user, nil
}

// SessionUser authenticates transport metadata using the same school check as the guard.
func (s *Service) SessionUser(ctx context.Context, authorization, schools []string) (models.User, error) {
	token, err := Bearer(authorization)
	if err != nil {
		return models.User{}, err
	}
	return s.scopedUser(ctx, token, schools)
}

func (g *Guard) User(target any) error {
	token, err := Bearer(g.Request.Header.Values("Authorization"))
	if err != nil {
		return err
	}
	u, err := g.current(token)
	if err != nil {
		return err
	}
	switch out := target.(type) {
	case *models.User:
		if out != nil {
			*out = u
			return nil
		}
	case *access.Principal:
		if out != nil {
			*out = access.Principal{ID: u.ID, School: u.Membership.School, Permissions: u.Permissions}
			return nil
		}
	}
	return errors.New("Auth.User requires a pointer to a user or principal")
}
func (g *Guard) Check() bool         { _, err := g.ID(); return err == nil }
func (g *Guard) Guest() bool         { return !g.Check() }
func (g *Guard) ID() (string, error) { var p access.Principal; err := g.User(&p); return p.ID, err }
func (g *Guard) Parse(token string) (*framework.Payload, error) {
	u, err := g.current(token)
	if err != nil {
		return nil, err
	}
	expiry := time.Unix(u.ExpiresAt, 0)
	return &framework.Payload{Key: u.ID, Guard: "school", ExpireAt: expiry, IssuedAt: expiry.Add(-sessionTTL)}, nil
}
func (g *Guard) Login(input any) (string, error) {
	c, ok := input.(Credentials)
	if !ok {
		return "", ErrInvalidCredentials
	}
	s, err := g.Resolve()
	if err != nil {
		return "", err
	}
	session, err := s.Login(g.Request.Context(), c.School, c.Email, c.Password)
	return session.Token, err
}
func (g *Guard) LoginUsingID(any) (string, error) { return "", ErrInvalidCredentials }
func (g *Guard) Refresh() (string, error) {
	return "", errors.New("sign in again to renew a school session")
}
func (g *Guard) Logout() error {
	token, err := Bearer(g.Request.Header.Values("Authorization"))
	if err != nil {
		return err
	}
	if _, err = g.current(token); err != nil {
		return err
	}
	s, err := g.Resolve()
	if err != nil {
		return err
	}
	return s.Logout(g.Request.Context(), token)
}
