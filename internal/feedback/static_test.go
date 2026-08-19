package feedback

import (
	"io"
	"net/http"
	"testing"
)

// property [1]: GET / returns 200 with the embedded index.html body.
func TestServesEmbeddedIndexAtRoot(t *testing.T) {
	srv, base := startTestServer(t, FeedbackRequest{ID: "s1", Surface: "canvas"})
	defer srv.stopForTest()

	resp, err := http.Get(base + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("GET / status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	want, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatalf("read embedded index.html: %v", err)
	}
	if string(body) != string(want) {
		t.Fatalf("GET / body = %q, want embedded index.html %q", body, want)
	}
}

// property [3]: GET /<asset> for an embedded non-index file returns 200 with
// that file's bytes.
func TestServesEmbeddedAsset(t *testing.T) {
	srv, base := startTestServer(t, FeedbackRequest{ID: "s1", Surface: "canvas"})
	defer srv.stopForTest()

	resp, err := http.Get(base + "/app.js")
	if err != nil {
		t.Fatalf("GET /app.js: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("GET /app.js status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	want, err := webFS.ReadFile("web/app.js")
	if err != nil {
		t.Fatalf("read embedded app.js: %v", err)
	}
	if string(body) != string(want) {
		t.Fatalf("GET /app.js body = %q, want embedded app.js %q", body, want)
	}
}
