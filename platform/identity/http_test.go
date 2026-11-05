package identity

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func call(t *testing.T, h http.Handler, method string, payload any, token, scope string) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, "/api/v1/session", bytes.NewReader(b))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if scope != "" {
		req.Header.Set("X-School-ID", scope)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestHTTPLoginAuthenticateScopeAndRevoke(t *testing.T) {
	s := fixture(t)
	password := secret(t)
	principal, err := s.Seed(t.Context(), "north", "demo@example.test", password)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(s)
	if w := call(t, h, "GET", nil, "", ""); w.Code != 401 {
		t.Fatalf("missing token: %d", w.Code)
	}
	if w := call(t, h, "POST", map[string]string{"unexpected": "field"}, "", ""); w.Code != 400 {
		t.Fatalf("unknown field: %d", w.Code)
	}
	if w := call(t, h, "POST", loginInput{"north", "demo@example.test", secret(t)}, "", ""); w.Code != 401 {
		t.Fatalf("bad password: %d", w.Code)
	}
	w := call(t, h, "POST", loginInput{"north", "demo@example.test", password}, "", "")
	if w.Code != 200 {
		t.Fatalf("login: %d", w.Code)
	}
	var session Session
	if err := json.Unmarshal(w.Body.Bytes(), &session); err != nil || session.Token == "" || session.Principal.ID != principal.ID {
		t.Fatalf("login envelope: %v", err)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("login response may be cached")
	}
	w = call(t, h, "GET", nil, session.Token, "north")
	if w.Code != 200 {
		t.Fatalf("read session: %d", w.Code)
	}
	var got Principal
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got.ID != principal.ID {
		t.Fatalf("GET principal: %v", err)
	}
	seen := false
	protected := s.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := FromContext(r.Context())
		if !ok || p.ID != principal.ID {
			t.Error("missing verified principal")
		}
		seen = true
		w.WriteHeader(204)
	}))
	if w := call(t, protected, "GET", nil, session.Token, "south"); w.Code != 403 || seen {
		t.Fatal("spoofed school reached downstream")
	}
	if w := call(t, protected, "GET", nil, session.Token, ""); w.Code != 204 || !seen {
		t.Fatal("verified caller failed to reach downstream")
	}
	if w := call(t, h, "DELETE", nil, session.Token, ""); w.Code != 204 {
		t.Fatalf("logout: %d", w.Code)
	}
	if w := call(t, h, "GET", nil, session.Token, ""); w.Code != 401 {
		t.Fatalf("revoked token: %d", w.Code)
	}
}

func TestLoginThrottle(t *testing.T) {
	h := NewHandler(fixture(t))
	for i := 0; i < 8; i++ {
		if w := call(t, h, "POST", loginInput{School: "north", Email: "missing@example.test"}, "", ""); w.Code != 401 {
			t.Fatalf("attempt %d = %d", i, w.Code)
		}
	}
	w := call(t, h, "POST", loginInput{School: "north", Email: "missing@example.test"}, "", "")
	if w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatalf("throttle response %d", w.Code)
	}
}
