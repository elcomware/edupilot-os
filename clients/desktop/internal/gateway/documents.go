package gateway

import (
	"mime"
	"net/http"
	"regexp"
	"strings"
)

const documentFiles = "/api/v1/documents/files"

// Kept in sync with Documents' MaxFileSize. This host is a separate Go module.
const maxDocumentBytes = 8 << 20

// Documents generates canonical lowercase RFC 4122 version 4 IDs.
var documentID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

var documentSuffix = map[string]string{
	"application/pdf": ".pdf",
	"image/png":       ".png",
	"image/jpeg":      ".jpg",
	"text/plain":      ".txt",
}

func documentPathID(path string) string {
	id, ok := strings.CutPrefix(path, documentFiles+"/")
	if !ok || !documentID.MatchString(id) {
		return ""
	}
	return id
}

// Set headers only after all fields pass validation. Else callers can return
// a JSON error without leaving attachment headers on the error response.
func setDownloadHeaders(dst, src http.Header, id string) bool {
	if len(src.Values("Content-Type")) != 1 || len(src.Values("Content-Disposition")) != 1 {
		return false
	}
	media, params, err := mime.ParseMediaType(src.Get("Content-Type"))
	suffix, allowed := documentSuffix[media]
	if err != nil || !allowed {
		return false
	}
	if len(params) > 0 && !(media == "text/plain" && len(params) == 1 && strings.EqualFold(params["charset"], "utf-8")) {
		return false
	}
	disposition, options, err := mime.ParseMediaType(src.Get("Content-Disposition"))
	filename := id + suffix
	if err != nil || disposition != "attachment" || len(options) != 1 || options["filename"] != filename {
		return false
	}
	dst.Set("Content-Type", mime.FormatMediaType(media, params))
	dst.Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	dst.Set("Content-Security-Policy", "sandbox; default-src 'none'")
	dst.Set("Cache-Control", "private, no-store")
	return true
}
