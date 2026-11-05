package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func TestOriginSeparatesDeploymentAndDevelopment(t *testing.T) {
	for _, raw := range []string{"https://school.example", "http://127.0.0.1:3000", "http://[::1]:3000", ""} {
		if _, err := Origin(raw); err != nil {
			t.Errorf("rejects allowed %q: %v", raw, err)
		}
	}
	for _, raw := range []string{"http://school.example", "https://school.example/path", "https://someone@school.example", "https://school.example/?secret", "file:///etc", "//school.example"} {
		if _, err := Origin(raw); err == nil {
			t.Errorf("accepts invalid %q", raw)
		}
	}
}

func TestForwardsCredentialsButNotTenantOverrideOrCookies(t *testing.T) {
	const student = "{\"givenName\":\"Test\",\"familyName\":\"Only\"}"
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() != "/api/v1/students?offset=0" || r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer fixture" {
			t.Error("request contract changed")
		}
		if r.Header.Get("Cookie") != "" || r.Header.Get("X-School-ID") != "" || r.Header.Get("X-Forwarded-For") != "" {
			t.Error("unexpected ambient identity")
		}
		b, err := io.ReadAll(r.Body)
		if err != nil || string(b) != student {
			t.Error("request body changed")
		}
		w.Header().Set("Set-Cookie", "bad=x")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(student))
	}))
	defer up.Close()
	u, _ := url.Parse(up.URL)
	proxy, closeIdle := New(u)
	defer closeIdle()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/students?offset=0", strings.NewReader(student))
	for k, v := range map[string]string{"Authorization": "Bearer fixture", "Cookie": "other=x", "X-School-ID": "other", "X-Forwarded-For": "bad"} {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	Mount(http.NotFoundHandler(), proxy).ServeHTTP(w, r)
	if w.Code != http.StatusCreated || w.Body.String() != student || w.Header().Get("Set-Cookie") != "" {
		t.Errorf("unexpected gateway response: %d %v", w.Code, w.Header())
	}
}

func TestRejectsUpstreamRedirectWithoutForwardingBearer(t *testing.T) {
	var calls atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls.Add(1) }))
	defer other.Close()
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, other.URL, 307) }))
	defer up.Close()
	u, _ := url.Parse(up.URL)
	proxy, closeIdle := New(u)
	defer closeIdle()
	r := httptest.NewRequest("GET", "/api/v1/session", nil)
	r.Header.Set("Authorization", "Bearer fixture")
	w := httptest.NewRecorder()
	Mount(http.NotFoundHandler(), proxy).ServeHTTP(w, r)
	if w.Code != http.StatusBadGateway || calls.Load() != 0 || w.Header().Get("Location") != "" {
		t.Fatal("upstream redirect escaped configured origin")
	}
}

func TestRejectsPathEscape(t *testing.T) {
	never := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("invalid path reached handler") })
	handler := Mount(never, never)
	for _, path := range []string{"/api/v1/../secrets", "/api/v1/students%2Ftransfer", "/api/v2/foo", "/api/v1/%252e%252e/private"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 404 {
			t.Errorf("%s: %d", path, w.Code)
		}
	}
}
