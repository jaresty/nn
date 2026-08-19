package feedback

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// property [6]: a prior draft is retrievable via GET /session/<id>/draft so the
// surface can restore it as initialData on reopen.
func TestGetDraftReturnsPersistedDraft(t *testing.T) {
	srv, base := startTestServer(t, FeedbackRequest{ID: "s1", Surface: "canvas"})
	defer srv.stopForTest()

	scene := []byte(`{"elements":[{"id":"a"}],"appState":{}}`)
	putReq, _ := http.NewRequest(http.MethodPut, base+"/session/s1/draft", bytes.NewReader(scene))
	if _, err := http.DefaultClient.Do(putReq); err != nil {
		t.Fatalf("PUT draft: %v", err)
	}

	resp, err := http.Get(base + "/session/s1/draft")
	if err != nil {
		t.Fatalf("GET draft: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("GET draft status = %d, want 200", resp.StatusCode)
	}
	body := make([]byte, len(scene))
	n, _ := resp.Body.Read(body)
	if !bytes.Equal(body[:n], scene) {
		t.Fatalf("GET draft body = %s, want %s", body[:n], scene)
	}
}

// property [8]: POST /session/<id>/png persists the exported png so submit can
// name it as a result.png artifact.
func TestPostPngPersists(t *testing.T) {
	srv, base := startTestServer(t, FeedbackRequest{ID: "s1", Surface: "canvas"})
	defer srv.stopForTest()

	png := []byte("\x89PNG\r\n\x1a\n fake png bytes")
	resp, err := http.Post(base+"/session/s1/png", "image/png", bytes.NewReader(png))
	if err != nil {
		t.Fatalf("POST png: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		t.Fatalf("POST png status = %d, want 2xx", resp.StatusCode)
	}
	got, err := os.ReadFile(filepath.Join(srv.dir, "result.png"))
	if err != nil {
		t.Fatalf("read result.png: %v", err)
	}
	if !bytes.Equal(got, png) {
		t.Fatalf("result.png = %d bytes, want %d", len(got), len(png))
	}
}

// property [8a]: when a png has been uploaded, submitting a canvas session names
// a result.png artifact in the envelope alongside the excalidraw artifact.
func TestCanvasSubmitNamesPngArtifactWhenPresent(t *testing.T) {
	srv, base := startTestServer(t, FeedbackRequest{ID: "s1", Surface: "canvas"})

	scene := []byte(`{"elements":[],"appState":{}}`)
	putReq, _ := http.NewRequest(http.MethodPut, base+"/session/s1/draft", bytes.NewReader(scene))
	http.DefaultClient.Do(putReq)
	http.Post(base+"/session/s1/png", "image/png", bytes.NewReader([]byte("png")))

	go srv.Wait()
	resp, err := http.Post(base+"/session/s1/submit", "application/json", nil)
	if err != nil {
		t.Fatalf("POST submit: %v", err)
	}
	resp.Body.Close()

	result, err := ReadResult(srv.dir)
	if err != nil {
		t.Fatalf("ReadResult: %v", err)
	}
	hasPng := false
	for _, a := range result.Artifacts {
		if a.Format == "png" && a.Path == "result.png" {
			hasPng = true
		}
	}
	if !hasPng {
		t.Fatalf("result artifacts %+v: no png artifact", result.Artifacts)
	}
}
