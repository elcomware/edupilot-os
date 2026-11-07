package access

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/elcomware/edupilot/modules/core/presentation"
)

func Bearer(values []string) (string, error) {
	if len(values) != 1 {
		return "", ErrUnauthenticated
	}
	fields := strings.Fields(values[0])
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return "", ErrUnauthenticated
	}
	return fields[1], nil
}

// ScopeQuery rejects duplicate or unknown parameters and ambiguous transport
// selectors. The credential realm remains X-School-ID; hierarchy uses tenant.
func ScopeQuery(r *http.Request, allowed ...string) (url.Values, int, error) {
	if len(r.URL.RawQuery) > 4096 || len(r.Header.Values("X-School-ID")) > 1 || len(r.Header.Values("Authorization")) > 1 || len(r.Header.Values("X-Tenant-ID")) > 0 {
		return nil, 0, ErrInvalidScope
	}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, 0, ErrInvalidScope
	}
	for key, v := range values {
		valid := false
		for _, a := range allowed {
			if key == a {
				valid = true
				break
			}
		}
		if !valid || len(v) != 1 || v[0] == "" {
			return nil, 0, ErrInvalidScope
		}
	}
	limit := 25
	if values.Has("limit") {
		limit, err = strconv.Atoi(values.Get("limit"))
		if err != nil || limit < 1 || limit > 100 {
			return nil, 0, ErrInvalidScope
		}
	}
	return values, limit, nil
}

func ScopeError(w http.ResponseWriter, err error) {
	status, code, message := 503, "unavailable", "Scoped access is unavailable."
	switch {
	case errors.Is(err, ErrUnauthenticated):
		status, code, message = 401, "unauthenticated", "Invalid or expired session."
	case errors.Is(err, ErrForbidden):
		status, code, message = 403, "forbidden", "Access denied for this tenant or action."
	case errors.Is(err, ErrInvalidScope):
		status, code, message = 422, "invalid_request", "Invalid scoped request."
	}
	ScopeHTTPError(w, status, code, message)
}
func ScopeHTTPError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Cache-Control", "no-store")
	if status == 401 {
		w.Header().Set("WWW-Authenticate", `Bearer realm="edupilot"`)
	}
	presentation.JSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
