// Package media identifies a media type from file content, independent of any request metadata.
package media

import (
	"mime"
	"net/http"
)

// Detect inspects the file signature using Go's standard content sniffer.
func Detect(data []byte) (string, error) {
	value, _, err := mime.ParseMediaType(http.DetectContentType(data))
	return value, err
}
