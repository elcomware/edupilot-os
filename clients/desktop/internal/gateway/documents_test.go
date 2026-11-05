package gateway

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const fixtureDocument = "4f06f3dc-bdb5-4fbe-9315-b31a3ca0e827"

func mountUpstream(t *testing.T, fn http.HandlerFunc) http.Handler {
	t.Helper()
	up := httptest.NewServer(fn)
	t.Cleanup(up.Close)
	u, err := url.Parse(up.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxy, closeIdle := New(u)
	t.Cleanup(closeIdle)
	return Mount(http.NotFoundHandler(), proxy)
}

func TestDocumentUploadAtExactLimit(t *testing.T) {
	data := bytes.Repeat([]byte{0xff, 0x00, 0x7f, 0x81}, maxDocumentBytes/4)
	path := documentFiles + "/" + fixtureDocument
	proxy := mountUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != documentFiles || r.Method != http.MethodPost || r.Header.Get("Content-Type") != "image/jpeg" || r.Header.Get("Authorization") != "Bearer fixture" {
			t.Error("upload request metadata changed")
		}
		received, err := io.ReadAll(r.Body)
		if err != nil || !bytes.Equal(received, data) {
			t.Error("binary upload changed")
		}
		w.Header().Set("Location", path)
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"`+fixtureDocument+`"}`)
	})
	r := httptest.NewRequest(http.MethodPost, documentFiles, bytes.NewReader(data))
	r.Header.Set("Content-Type", "image/jpeg")
	r.Header.Set("Authorization", "Bearer fixture")
	w := httptest.NewRecorder()
	proxy.ServeHTTP(w, r)
	if w.Code != 201 || w.Header().Get("Location") != path || w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("upload response %d %v", w.Code, w.Header())
	}
}

func TestRequestLimitsApplyBeforeForwarding(t *testing.T) {
	proxy := mountUpstream(t, func(http.ResponseWriter, *http.Request) { t.Error("oversized request reached school API") })
	for _, c := range []struct {
		path        string
		size        int
		knownLength bool
	}{
		{documentFiles, maxDocumentBytes + 1, true}, {documentFiles, maxDocumentBytes + 1, false},
		{"/api/v1/students", maxRequest + 1, false}, {documentFiles + "/" + fixtureDocument, maxRequest + 1, true},
	} {
		r := httptest.NewRequest(http.MethodPost, c.path, bytes.NewReader(make([]byte, c.size)))
		if !c.knownLength {
			r.ContentLength = -1
		}
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, r)
		if w.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("%s known-length=%v: status %d", c.path, c.knownLength, w.Code)
		}
	}
}

func TestPrivateFileDownloadAtExactLimit(t *testing.T) {
	data := bytes.Repeat([]byte{0xff, 0x00, 0x7f, 0x81}, maxDocumentBytes/4)
	disposition := `attachment; filename="` + fixtureDocument + `.png"`
	proxy := mountUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Disposition", disposition)
		w.Header().Set("Set-Cookie", "private=x")
		_, _ = w.Write(data)
	})
	w := httptest.NewRecorder()
	proxy.ServeHTTP(w, httptest.NewRequest(http.MethodGet, documentFiles+"/"+fixtureDocument, nil))
	if w.Code != 200 || !bytes.Equal(data, w.Body.Bytes()) || w.Header().Get("Content-Type") != "image/png" || w.Header().Get("Content-Disposition") != disposition || w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Content-Security-Policy") != "sandbox; default-src 'none'" || w.Header().Get("Set-Cookie") != "" {
		t.Errorf("invalid download response: %d %v", w.Code, w.Header())
	}
}

func TestOversizeOrUnsafeDownloadReturnsJSONError(t *testing.T) {
	const good = `attachment; filename="` + fixtureDocument + `.pdf"`
	for _, c := range []struct {
		name, media, disposition string
		size                     int
	}{
		{"oversized", "application/pdf", good, maxDocumentBytes + 1},
		{"unlisted MIME", "text/html", good, 1}, {"inline", "application/pdf", strings.Replace(good, "attachment", "inline", 1), 1},
		{"path traversal", "application/pdf", `attachment; filename="../` + fixtureDocument + `.pdf"`, 1}, {"wrong file type", "image/png", good, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			proxy := mountUpstream(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", c.media)
				w.Header().Set("Content-Disposition", c.disposition)
				_, _ = w.Write(bytes.Repeat([]byte("_"), c.size))
			})
			w := httptest.NewRecorder()
			proxy.ServeHTTP(w, httptest.NewRequest(http.MethodGet, documentFiles+"/"+fixtureDocument, nil))
			if w.Code != 502 || w.Header().Get("Content-Type") != "application/json" || w.Header().Get("Content-Disposition") != "" {
				t.Errorf("unsafe response: %d %v", w.Code, w.Header())
			}
		})
	}
}

func TestFileErrorsStayJSONAndExternalUploadLocationIsDiscarded(t *testing.T) {
	proxy := mountUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Location", "https://other.example"+documentFiles+"/"+fixtureDocument)
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":"`+fixtureDocument+`"}`)
			return
		}
		w.Header().Set("Content-Disposition", "inline")
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"error":{"code":"forbidden"}}`)
	})
	w := httptest.NewRecorder()
	proxy.ServeHTTP(w, httptest.NewRequest(http.MethodGet, documentFiles+"/"+fixtureDocument, nil))
	if w.Code != 403 || w.Header().Get("Content-Disposition") != "" || w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("invalid file error: %d %v", w.Code, w.Header())
	}
	w = httptest.NewRecorder()
	proxy.ServeHTTP(w, httptest.NewRequest(http.MethodPost, documentFiles, strings.NewReader("x")))
	if w.Code != 201 || w.Header().Get("Location") != "" {
		t.Errorf("unsafe upload location: %d %v", w.Code, w.Header())
	}
}
