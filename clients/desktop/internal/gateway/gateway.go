// Package gateway forwards versioned API requests to one operator-configured
// backend. There is no listener, database, tenant selector or session authority here.
package gateway

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const (
	maxRequest  = 1 << 20
	maxResponse = 8 << 20
)

func Origin(raw string) (*url.URL, error) {
	if raw == "" {
		raw = "http://127.0.0.1:3000"
	}
	u, err := url.Parse(raw)
	if err != nil || !u.IsAbs() || u.Hostname() == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return nil, errors.New("EDUPILOT_API_ORIGIN must be an HTTP(S) origin without credentials, path, query or fragment")
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback()))) {
		return nil, errors.New("EDUPILOT_API_ORIGIN requires HTTPS except on loopback")
	}
	return &url.URL{Scheme: u.Scheme, Host: u.Host}, nil
}

// Mount routes only literal versioned API paths to the remote server.
func Mount(assets, api http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if !strings.HasPrefix(r.URL.Path, "/api/v1/") || path.Clean(r.URL.Path) != r.URL.Path || strings.ContainsAny(r.URL.Path, "\\%") || strings.Contains(strings.ToLower(r.URL.EscapedPath()), "%2f") {
				fail(w, http.StatusNotFound, "not_found")
				return
			}
			api.ServeHTTP(w, r)
			return
		}
		assets.ServeHTTP(w, r)
	})
}

func fail(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	fmt.Fprintf(w, "{\"error\":{\"code\":%q,\"message\":%q}}", code, http.StatusText(status))
}

// New uses no environment HTTP proxy and never follows redirects or copies
// cookies, Host, X-Forwarded-* or school-selection headers from the webview.
func New(origin *url.URL) (http.Handler, func()) {
	tr := &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout: 8 * time.Second, ResponseHeaderTimeout: 50 * time.Second,
		MaxResponseHeaderBytes: 64 << 10, IdleConnTimeout: 60 * time.Second, ForceAttemptHTTP2: true,
	}
	client := &http.Client{Transport: tr, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	documentTransfers := make(chan struct{}, 8)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
		default:
			fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}
		upload := r.Method == http.MethodPost && r.URL.Path == documentFiles
		var download string
		if r.Method == http.MethodGet {
			download = documentPathID(r.URL.Path)
		}
		requestLimit, sizeCode := maxRequest, "request_too_large"
		if upload {
			requestLimit, sizeCode = maxDocumentBytes, "file_too_large"
		}
		deadline := 25 * time.Second
		if upload || download != "" {
			// Match the backend's bounded concurrency before buffering a file.
			select {
			case documentTransfers <- struct{}{}:
				defer func() { <-documentTransfers }()
			default:
				fail(w, http.StatusServiceUnavailable, "busy")
				return
			}
			deadline = 50 * time.Second // Documents allows up to 45 s for storage.
		}
		if r.ContentLength > int64(requestLimit) {
			fail(w, http.StatusRequestEntityTooLarge, sizeCode)
			return
		}
		var body io.Reader
		if r.Body != nil && r.Body != http.NoBody {
			data, err := io.ReadAll(io.LimitReader(r.Body, int64(requestLimit)+1))
			if err != nil {
				fail(w, http.StatusBadRequest, "invalid_body")
				return
			}
			if len(data) > requestLimit {
				fail(w, http.StatusRequestEntityTooLarge, sizeCode)
				return
			}
			body = bytes.NewReader(data)
		}
		dest := *origin
		dest.Path, dest.RawQuery = r.URL.Path, r.URL.RawQuery
		ctx, cancel := context.WithTimeout(r.Context(), deadline)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, r.Method, dest.String(), body)
		if err != nil {
			fail(w, http.StatusBadRequest, "invalid_request")
			return
		}
		for _, name := range []string{"Authorization", "Content-Type", "Accept", "Accept-Language"} {
			for _, value := range r.Header.Values(name) {
				req.Header.Add(name, value)
			}
		}
		res, err := client.Do(req)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				fail(w, http.StatusGatewayTimeout, "upstream_timeout")
				return
			}
			fail(w, http.StatusBadGateway, "upstream_unavailable")
			return
		}
		defer res.Body.Close()
		if res.StatusCode >= 300 && res.StatusCode < 400 {
			fail(w, http.StatusBadGateway, "upstream_redirect")
			return
		}
		responseLimit := maxResponse
		isDownload := download != "" && res.StatusCode == http.StatusOK
		if isDownload {
			responseLimit = maxDocumentBytes
		}
		if res.ContentLength > int64(responseLimit) {
			fail(w, http.StatusBadGateway, "invalid_upstream_body")
			return
		}
		data, err := io.ReadAll(io.LimitReader(res.Body, int64(responseLimit)+1))
		if err != nil || len(data) > responseLimit {
			fail(w, http.StatusBadGateway, "invalid_upstream_body")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if isDownload && !setDownloadHeaders(w.Header(), res.Header, download) {
			fail(w, http.StatusBadGateway, "invalid_upstream_attachment")
			return
		}
		if upload && res.StatusCode == http.StatusCreated {
			// A successful upload remains successful even if the upstream
			// omitted Location. Never pass through an external URL.
			location := res.Header.Get("Location")
			if len(res.Header.Values("Location")) == 1 && documentPathID(location) != "" {
				w.Header().Set("Location", location)
			}
		}
		w.WriteHeader(res.StatusCode)
		if res.StatusCode != http.StatusNoContent {
			_, _ = w.Write(data)
		}
	}), tr.CloseIdleConnections
}
