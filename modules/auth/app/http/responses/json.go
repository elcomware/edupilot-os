package responses

import (
	"errors"
	"net/http"

	"github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/modules/core/access"
	"github.com/elcomware/edupilot/modules/core/presentation"
)

func Error(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Cache-Control", "no-store")
	if status == 401 {
		w.Header().Set("WWW-Authenticate", `Bearer realm="edupilot"`)
	}
	presentation.JSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func ServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidCredentials):
		Error(w, 401, "invalid_credentials", "Invalid school, email or password.")
	case errors.Is(err, access.ErrUnauthenticated):
		Error(w, 401, "unauthenticated", "Invalid or expired session.")
	case errors.Is(err, access.ErrForbidden):
		Error(w, 403, "forbidden", "Access denied for this school or action.")
	default:
		Error(w, 500, "internal_error", "Unable to process request.")
	}
}
