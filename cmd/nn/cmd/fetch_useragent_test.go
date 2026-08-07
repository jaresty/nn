package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchSendsUserAgent(t *testing.T) {
	// property [1a]: runFetch sends a non-empty User-Agent on its outbound HTTP request
	var receivedUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><p>hello</p></body></html>`))
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	_ = runFetch(srv.URL, false, &stdout, &stderr, nil)

	if receivedUA == "" {
		t.Fatal("property [1a]: runFetch sent no User-Agent header")
	}
	if receivedUA == "Go-http-client/1.1" || receivedUA == "Go-http-client/2.0" {
		t.Fatalf("property [1a]: runFetch sent default Go UA %q — must send identifying nn UA", receivedUA)
	}
}
