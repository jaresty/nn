package feedback

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestEndToEndCanvasLoop exercises the full canvas session the way the browser
// does: fetch the served UI bundle at /, load the session, save a draft scene,
// upload a png, submit, and confirm the native artifacts land in the envelope.
func TestEndToEndCanvasLoop(t *testing.T) {
	srv, base := startTestServer(t, FeedbackRequest{
		ID: "s1", Surface: "canvas", Mode: "create", Instructions: "sketch the flow",
	})

	// The browser first loads / — assert the real Excalidraw bundle is served.
	rootResp, err := http.Get(base + "/?session=s1")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	rootBody, _ := io.ReadAll(rootResp.Body)
	rootResp.Body.Close()
	if !strings.Contains(string(rootBody), "Excalidraw") {
		t.Fatalf("served / does not contain the Excalidraw bundle (len %d)", len(rootBody))
	}

	// Load the session request.
	if r, err := http.Get(base + "/session/s1"); err != nil {
		t.Fatalf("GET session: %v", err)
	} else {
		r.Body.Close()
	}

	// onChange -> PUT draft (the Excalidraw scene).
	scene := []byte(`{"type":"excalidraw","elements":[{"id":"a","type":"rectangle"}],"appState":{}}`)
	putReq, _ := http.NewRequest(http.MethodPut, base+"/session/s1/draft", bytes.NewReader(scene))
	if _, err := http.DefaultClient.Do(putReq); err != nil {
		t.Fatalf("PUT draft: %v", err)
	}

	// Done -> POST png, then POST submit.
	http.Post(base+"/session/s1/png", "image/png", bytes.NewReader([]byte("\x89PNG fake")))

	go srv.Wait()
	if r, err := http.Post(base+"/session/s1/submit", "application/json", nil); err != nil {
		t.Fatalf("POST submit: %v", err)
	} else {
		r.Body.Close()
	}

	result, err := ReadResult(srv.dir)
	if err != nil {
		t.Fatalf("ReadResult: %v", err)
	}
	formats := map[string]string{}
	for _, a := range result.Artifacts {
		formats[a.Format] = a.Path
	}
	if formats["excalidraw"] != "result.excalidraw" {
		t.Fatalf("missing excalidraw artifact, got %+v", result.Artifacts)
	}
	if formats["png"] != "result.png" {
		t.Fatalf("missing png artifact, got %+v", result.Artifacts)
	}
	if result.Status != "submitted" {
		t.Fatalf("status = %q, want submitted", result.Status)
	}
}
