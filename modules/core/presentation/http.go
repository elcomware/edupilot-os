// Package presentation selects the HTTP representation at the platform boundary.
package presentation

import (
	"encoding/json"
	"io/fs"
	"math"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// PageOrJSON keeps versioned API URLs JSON-only. Web URLs default to the
// compiled React shell. Only the JSON path evaluates the protected query.
func PageOrJSON(w http.ResponseWriter, r *http.Request, web fs.FS, data http.HandlerFunc) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if strings.HasPrefix(r.URL.Path, "/api/") {
		data(w, r)
		return
	}
	w.Header().Add("Vary", "Accept")
	accept := strings.Join(r.Header.Values("Accept"), ",")
	html, jsonQ := quality(accept, "text/html"), quality(accept, "application/json")
	switch {
	case html == 0 && jsonQ == 0:
		w.WriteHeader(http.StatusNotAcceptable)
	case jsonQ > html:
		data(w, r)
	default:
		page, err := fs.ReadFile(web, "index.html")
		if err != nil {
			http.Error(w, "Web client unavailable.", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	}
}

// A more specific range overrides a wildcard, including an explicit q=0.
func quality(accept, offer string) float64 {
	if strings.TrimSpace(accept) == "" {
		return 1
	}
	best, result := -1, 0.0
	for part := range strings.SplitSeq(accept, ",") {
		media, params, err := mime.ParseMediaType(strings.TrimSpace(part))
		if err != nil {
			continue
		}
		rank := 0
		switch media {
		case offer:
			rank = 2
		case strings.SplitN(offer, "/", 2)[0] + "/*":
			rank = 1
		case "*/*":
		default:
			continue
		}
		if rank < best {
			continue
		}
		q := 1.0
		if raw, ok := params["q"]; ok {
			q, err = strconv.ParseFloat(raw, 64)
			if err != nil || math.IsNaN(q) || q < 0 || q > 1 {
				continue
			}
		}
		if rank > best || q > result {
			best, result = rank, q
		}
	}
	return result
}
