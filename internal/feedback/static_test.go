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

// property [3] (revised): the static route serves the embedded bundle and
// returns 404 for paths with no corresponding embedded file. The bundle is a
// single self-contained index.html (Vite singlefile build), so there are no
// separate asset files; the delivered behavior is index-or-404.
func TestServesNotFoundForUnknownAsset(t *testing.T) {
	srv, base := startTestServer(t, FeedbackRequest{ID: "s1", Surface: "canvas"})
	defer srv.stopForTest()

	resp, err := http.Get(base + "/does-not-exist.js")
	if err != nil {
		t.Fatalf("GET unknown asset: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("GET /does-not-exist.js status = %d, want 404", resp.StatusCode)
	}
}
