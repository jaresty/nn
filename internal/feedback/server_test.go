package feedback

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// startTestServer creates a session dir with a stored request and starts a
// Server for it, returning the server and its base URL. The caller must ensure
// the server is stopped via submit/cancel.
func startTestServer(t *testing.T, q FeedbackRequest) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	if err := WriteRequest(dir, q); err != nil {
		t.Fatalf("WriteRequest: %v", err)
	}
	srv, err := NewServer(q.ID, dir)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return srv, "http://" + srv.Addr()
}

func TestServerBindsLoopbackEphemeralPort(t *testing.T) {
	srv, base := startTestServer(t, FeedbackRequest{ID: "s1", Surface: "canvas"})
	defer srv.stopForTest()
	if !strings.HasPrefix(srv.Addr(), "127.0.0.1:") {
		t.Fatalf("Addr = %q, want 127.0.0.1: prefix", srv.Addr())
	}
	if strings.HasSuffix(srv.Addr(), ":0") {
		t.Fatalf("Addr = %q, want a concrete non-zero port", srv.Addr())
	}
	resp, err := http.Get(base + "/session/s1")
	if err != nil {
		t.Fatalf("GET unreachable: %v", err)
	}
	resp.Body.Close()
}

func TestGetSessionReturnsRequest(t *testing.T) {
	q := FeedbackRequest{ID: "s1", Surface: "canvas", Mode: "create", Instructions: "hi"}
	srv, base := startTestServer(t, q)
	defer srv.stopForTest()

	resp, err := http.Get(base + "/session/s1")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got FeedbackRequest
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != q.ID || got.Surface != q.Surface || got.Instructions != q.Instructions {
		t.Fatalf("GET body = %+v, want %+v", got, q)
	}
}

func TestPutDraftPersists(t *testing.T) {
	srv, base := startTestServer(t, FeedbackRequest{ID: "s1"})
	defer srv.stopForTest()

	payload := []byte(`{"elements":["a","b"]}`)
	req, _ := http.NewRequest(http.MethodPut, base+"/session/s1/draft", bytes.NewReader(payload))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		t.Fatalf("status = %d, want 2xx", resp.StatusCode)
	}
	got, err := os.ReadFile(filepath.Join(srv.dir, "draft.json"))
	if err != nil {
		t.Fatalf("read draft.json: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("draft.json = %s, want %s", got, payload)
	}
}

func TestSubmitWritesResultAndUnblocks(t *testing.T) {
	srv, base := startTestServer(t, FeedbackRequest{ID: "s1", Surface: "canvas"})

	// seed a draft so submit has something to promote
	payload := []byte(`{"artifacts":[{"format":"excalidraw","path":"x"}]}`)
	putReq, _ := http.NewRequest(http.MethodPut, base+"/session/s1/draft", bytes.NewReader(payload))
	if _, err := http.DefaultClient.Do(putReq); err != nil {
		t.Fatalf("PUT: %v", err)
	}

	outcomeCh := make(chan Outcome, 1)
	go func() { outcomeCh <- srv.Wait() }()

	resp, err := http.Post(base+"/session/s1/submit", "application/json", nil)
	if err != nil {
		t.Fatalf("POST submit: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		t.Fatalf("status = %d, want 2xx", resp.StatusCode)
	}

	outcome := <-outcomeCh
	if outcome != OutcomeSubmitted {
		t.Fatalf("outcome = %q, want %q", outcome, OutcomeSubmitted)
	}
	if _, err := os.Stat(filepath.Join(srv.dir, "result.json")); err != nil {
		t.Fatalf("result.json not written: %v", err)
	}
}

func TestCancelUnblocks(t *testing.T) {
	srv, base := startTestServer(t, FeedbackRequest{ID: "s1"})

	outcomeCh := make(chan Outcome, 1)
	go func() { outcomeCh <- srv.Wait() }()

	resp, err := http.Post(base+"/session/s1/cancel", "application/json", nil)
	if err != nil {
		t.Fatalf("POST cancel: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		t.Fatalf("status = %d, want 2xx", resp.StatusCode)
	}
	if outcome := <-outcomeCh; outcome != OutcomeCancelled {
		t.Fatalf("outcome = %q, want %q", outcome, OutcomeCancelled)
	}
}
