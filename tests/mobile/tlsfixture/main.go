// The hosted mobile fixture terminates TLS on loopback for a disposable school API.
package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("expected fixture certificate and key paths")
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(&url.URL{Scheme: "http", Host: "127.0.0.1:3000"})
		},
		ModifyResponse: func(r *http.Response) error {
			// Only record the result of session creation. Never log request or response bodies or headers.
			if r.Request.Method == http.MethodPost && r.Request.URL.Path == "/api/v1/session" {
				log.Printf("Mobile fixture session POST returned %d", r.StatusCode)
			}
			return nil
		},
	}
	server := http.Server{Addr: "127.0.0.1:4443", Handler: proxy, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(server.ListenAndServeTLS(os.Args[1], os.Args[2]))
}
