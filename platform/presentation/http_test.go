package presentation

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestWebAndAPIRepresentations(t *testing.T) {
	const page = "<!doctype html><div id=root></div>"
	web := fstest.MapFS{"index.html": {Data: []byte(page)}}
	for _, tc := range []struct {
		path, accept string
		status       int
		content      string
	}{
		{"/login", "", 200, "text/html"}, {"/login", "*/*", 200, "text/html"},
		{"/login", "application/json", 200, "application/json"},
		{"/login", "application/json;q=0.8, text/html;q=0.5", 200, "application/json"},
		{"/login", "text/*;q=0, */*", 200, "application/json"},
		{"/login", "application/json;q=0, */*", 200, "text/html"},
		{"/login", "text/html;q=0, application/json;q=0", 406, ""},
		{"/login", "image/png", 406, ""},
		{"/api/v1/session", "text/html", 200, "application/json"},
	} {
		r := httptest.NewRequest("GET", tc.path, nil)
		r.Header.Set("Accept", tc.accept)
		w := httptest.NewRecorder()
		queried := false
		PageOrJSON(w, r, web, func(w http.ResponseWriter, _ *http.Request) { queried = true; JSON(w, 200, "session") })
		if w.Code != tc.status || !strings.HasPrefix(w.Header().Get("Content-Type"), tc.content) || queried != (tc.content == "application/json") || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s %q: %d %v", tc.path, tc.accept, w.Code, w.Header())
		}
		if tc.path == "/login" && w.Header().Get("Vary") != "Accept" {
			t.Fatal("missing variant header")
		}
		if tc.content == "text/html" && w.Body.String() != page {
			t.Fatal("wrong shell")
		}
	}
	w := httptest.NewRecorder()
	PageOrJSON(w, httptest.NewRequest("GET", "/login", nil), fstest.MapFS{}, nil)
	if w.Code != 503 {
		t.Fatalf("missing bundle: %d", w.Code)
	}
}
