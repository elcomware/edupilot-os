package main

import (
	"bytes"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Exercise the compiled CLI against signed HTTPS requests on a disposable
// protocol fixture, using the exact record previously accepted by local recovery.
func exerciseRestoredBucket(t *testing.T, root, id, manifest, references, digest string, expected []byte) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "tenant-document-inventory")
	if out, err := exec.Command("go", "build", "-buildvcs=false", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	key := "schools/north/documents/" + id
	record, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(key)))
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"restored", "missing", "unexpected-next-page", "corrupt", "changed-etag", "wrong-baseline"} {
		t.Run(scenario, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
					t.Error("unexpected method or unsigned request")
					w.WriteHeader(403)
					return
				}
				if r.URL.Query().Get("list-type") == "2" && strings.TrimRight(r.URL.Path, "/") == "/recovery" {
					entry := fmt.Sprintf("<Contents><Key>%s</Key><ETag>\"fixture\"</ETag><Size>%d</Size></Contents>", key, len(record))
					page := ""
					switch scenario {
					case "missing":
						entry = ""
					case "unexpected-next-page":
						if r.URL.Query().Get("continuation-token") == "" {
							page = "<IsTruncated>true</IsTruncated><NextContinuationToken>next</NextContinuationToken>"
						} else {
							entry = strings.Replace(entry, key, "foreign", 1)
						}
					}
					fmt.Fprintf(w, "<ListBucketResult>%s%s</ListBucketResult>", page, entry)
					return
				}
				if r.URL.Path != "/recovery/"+key || r.Header.Get("If-Match") != "\"fixture\"" {
					t.Error("unexpected object read")
					w.WriteHeader(403)
					return
				}
				if scenario == "changed-etag" {
					w.WriteHeader(412)
					return
				}
				w.Header().Set("ETag", "\""+"fixture"+"\"")
				w.Header().Set("Last-Modified", time.Unix(1, 0).UTC().Format(http.TimeFormat))
				w.Header().Set("Content-Length", strconv.Itoa(len(record)))
				body := bytes.Clone(record)
				if scenario == "corrupt" {
					body[len(body)-1] ^= 1
				}
				w.Write(body)
			}))
			defer server.Close()
			ca := filepath.Join(t.TempDir(), "ca.pem")
			if err := os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600); err != nil {
				t.Fatal(err)
			}
			approved := digest
			if scenario == "wrong-baseline" {
				approved = strings.Repeat("0", 64)
			}
			cmd := exec.Command(bin, "-s3-restore", "-manifest", manifest, "-references", references, "-expected-sha256", approved)
			cmd.Env = append(os.Environ(), "SSL_CERT_FILE="+ca, "EDUPILOT_RESTORE_S3_ENDPOINT="+server.URL, "EDUPILOT_RESTORE_S3_BUCKET=recovery", "EDUPILOT_RESTORE_S3_ACCESS_KEY_ID=test", "EDUPILOT_RESTORE_S3_SECRET_ACCESS_KEY=test-secret")
			var diagnostics bytes.Buffer
			cmd.Stderr = &diagnostics
			out, err := cmd.Output()
			if scenario == "restored" {
				if err != nil || !bytes.Equal(out, expected) {
					t.Fatalf("restored report differs: %v %s", err, diagnostics.String())
				}
			} else if err == nil || len(out) != 0 {
				t.Fatal("unverified bucket produced a report")
			}
			if strings.Contains(diagnostics.String(), id) || strings.Contains(diagnostics.String(), "test-secret") {
				t.Fatal("diagnostics disclosed private data")
			}
		})
	}
}
