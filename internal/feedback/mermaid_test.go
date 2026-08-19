package feedback

import (
	"encoding/json"
	"net/http"
	"testing"
)

// property [9a]: FeedbackRequest round-trips a Mermaid field through disk.
func TestRequestRoundTripsMermaid(t *testing.T) {
	dir := t.TempDir()
	q := FeedbackRequest{ID: "s1", Surface: "canvas", Mermaid: "graph TD; A-->B"}
	if err := WriteRequest(dir, q); err != nil {
		t.Fatalf("WriteRequest: %v", err)
	}
	got, err := ReadRequest(dir)
	if err != nil {
		t.Fatalf("ReadRequest: %v", err)
	}
	if got.Mermaid != q.Mermaid {
		t.Fatalf("Mermaid round-trip = %q, want %q", got.Mermaid, q.Mermaid)
	}
}

// property [9b]: GET /session/<id> includes the Mermaid field so the surface
// can seed the canvas from it.
func TestGetSessionIncludesMermaid(t *testing.T) {
	q := FeedbackRequest{ID: "s1", Surface: "canvas", Mermaid: "graph TD; A-->B"}
	srv, base := startTestServer(t, q)
	defer srv.stopForTest()

	resp, err := http.Get(base + "/session/s1")
	if err != nil {
		t.Fatalf("GET session: %v", err)
	}
	defer resp.Body.Close()
	var got FeedbackRequest
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Mermaid != q.Mermaid {
		t.Fatalf("GET session Mermaid = %q, want %q", got.Mermaid, q.Mermaid)
	}
}
