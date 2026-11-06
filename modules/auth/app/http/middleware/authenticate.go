package middleware

import (
	"net/http"

	"github.com/elcomware/edupilot/modules/auth/app/http/responses"
	"github.com/elcomware/edupilot/modules/core/access"
)

// ServeAuthenticated forwards only the principal verified by the request's Goravel guard.
func ServeAuthenticated(w http.ResponseWriter, r *http.Request, next http.Handler, guard interface{ User(any) error }) {
	var principal access.Principal
	if err := guard.User(&principal); err != nil {
		responses.ServiceError(w, err)
		return
	}
	next.ServeHTTP(w, r.WithContext(access.WithPrincipal(r.Context(), principal)))
}
