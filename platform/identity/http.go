package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Bearer requires a single, unambiguous Authorization header.
func Bearer(r *http.Request) (string, error) {
	values := r.Header.Values("Authorization")
	if len(values) != 1 {
		return "", ErrUnauthenticated
	}
	fields := strings.Fields(values[0])
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return "", ErrUnauthenticated
	}
	if _, err := digest(fields[1]); err != nil {
		return "", ErrUnauthenticated
	}
	return fields[1], nil
}

func checkScope(r *http.Request, p Principal) error {
	values := r.Header.Values("X-School-ID")
	if len(values) > 0 && (len(values) != 1 || values[0] != p.School) {
		return ErrForbidden
	}
	return nil
}

func (s *Service) requestPrincipal(r *http.Request) (Principal, error) {
	token, err := Bearer(r)
	if err != nil {
		return Principal{}, err
	}
	p, err := s.Authenticate(r.Context(), token)
	if err != nil {
		return Principal{}, err
	}
	if err = checkScope(r, p); err != nil {
		return Principal{}, err
	}
	return p, nil
}

// Protect authenticates every request and carries only the verified principal into the handler.
func (s *Service) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := s.requestPrincipal(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), p)))
	})
}

type sessionHandler struct {
	service  *Service
	attempts *attemptWindow
	checking chan struct{}
}

// NewHandler should be constructed once per server. The source/identity limits
// and hash concurrency cap are process-local; distributed deployments should
// additionally enforce ingress-wide limits.
func NewHandler(s *Service) http.Handler {
	return &sessionHandler{service: s, attempts: newAttemptWindow(), checking: make(chan struct{}, 8)}
}

type loginInput struct {
	School   string `json:"school"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *sessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	switch r.Method {
	case http.MethodPost:
		// Trust the socket peer, never an arbitrary forwarded address header.
		source, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			source = r.RemoteAddr
		}
		now := h.service.now()
		if !h.attempts.allow("ip:"+source, 30, now) {
			throttled(w)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8192)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		var in loginInput
		if err := dec.Decode(&in); err != nil || dec.Decode(new(any)) != io.EOF {
			writeError(w, 400, "invalid_request", "Expected a JSON object with school, email and password.")
			return
		}
		key := sha256.Sum256([]byte(in.School + "\x00" + strings.ToLower(strings.TrimSpace(in.Email))))
		if !h.attempts.allow("account:"+hex.EncodeToString(key[:]), 8, now) {
			throttled(w)
			return
		}
		select {
		case h.checking <- struct{}{}:
			defer func() { <-h.checking }()
		default:
			throttled(w)
			return
		}
		session, err := h.service.Login(r.Context(), in.School, in.Email, in.Password)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, 200, session)
	case http.MethodGet:
		p, err := h.service.requestPrincipal(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, 200, p)
	case http.MethodDelete:
		if _, err := h.service.requestPrincipal(r); err != nil {
			writeServiceError(w, err)
			return
		}
		token, err := Bearer(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		if err = h.service.Logout(r.Context(), token); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "POST, GET, DELETE")
		writeError(w, 405, "method_not_allowed", "Method not allowed.")
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Cache-Control", "no-store")
	if status == 401 {
		w.Header().Set("WWW-Authenticate", `Bearer realm="edupilot"`)
	}
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		writeError(w, 401, "invalid_credentials", "Invalid school, email or password.")
	case errors.Is(err, ErrUnauthenticated):
		writeError(w, 401, "unauthenticated", "Invalid or expired session.")
	case errors.Is(err, ErrForbidden):
		writeError(w, 403, "forbidden", "Access denied for this school or action.")
	default:
		writeError(w, 500, "internal_error", "Unable to process request.")
	}
}

func throttled(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "60")
	writeError(w, 429, "rate_limited", "Too many sign-in attempts. Try again later.")
}

type windowEntry struct {
	n       int
	expires time.Time
}
type attemptWindow struct {
	mu         sync.Mutex
	entries    map[string]windowEntry
	cleanAfter time.Time
}

func newAttemptWindow() *attemptWindow { return &attemptWindow{entries: make(map[string]windowEntry)} }

// Bound state even with attacker-selected account names or many peer addresses.
// When at capacity, refuse new keys until eligible entries expire.
func (a *attemptWindow) allow(key string, limit int, now time.Time) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !now.Before(a.cleanAfter) {
		for k, v := range a.entries {
			if !now.Before(v.expires) {
				delete(a.entries, k)
			}
		}
		a.cleanAfter = now.Add(5 * time.Second)
	}
	v, ok := a.entries[key]
	if !ok || !now.Before(v.expires) {
		if !ok && len(a.entries) >= 8192 {
			return false
		}
		v = windowEntry{expires: now.Add(time.Minute)}
	}
	if v.n >= limit {
		return false
	}
	v.n++
	a.entries[key] = v
	return true
}
