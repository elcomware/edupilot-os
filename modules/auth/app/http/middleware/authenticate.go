package middleware

import (
	"context"
	"github.com/elcomware/edupilot/modules/auth/app/http/responses"
	"github.com/elcomware/edupilot/modules/auth/app/models"
	"github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/platform/access"
	"net/http"
	"strings"
)

// Bearer requires a single, unambiguous Authorization header.
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

func CheckScope(values []string, school string) error {
	if len(values) > 0 && (len(values) != 1 || values[0] != school) {
		return access.ErrForbidden
	}
	return nil
}

func RequestPrincipal(s *services.Service, r *http.Request) (access.Principal, error) {
	token, err := Bearer(r.Header.Values("Authorization"))
	if err != nil {
		return access.Principal{}, err
	}
	p, err := s.Authenticate(r.Context(), token)
	if err != nil {
		return access.Principal{}, err
	}
	if err = CheckScope(r.Header.Values("X-School-ID"), p.School); err != nil {
		return access.Principal{}, err
	}
	return p, nil
}

// Protect authenticates every request and carries only the verified principal into the handler.
func Protect(s *services.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := RequestPrincipal(s, r)
		if err != nil {
			responses.ServiceError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(access.WithPrincipal(r.Context(), p)))
	})
}

func SessionUser(ctx context.Context, s *services.Service, authorization, schools []string) (models.User, error) {
	token, err := Bearer(authorization)
	if err != nil {
		return models.User{}, err
	}
	user, err := s.CurrentUser(ctx, token)
	if err != nil {
		return models.User{}, err
	}
	if err = CheckScope(schools, user.Membership.School); err != nil {
		return models.User{}, err
	}
	return user, nil
}
