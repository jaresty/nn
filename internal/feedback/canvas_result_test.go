package feedback

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// property [7a] + [7b]: submitting a canvas session promotes the draft (the
// Excalidraw scene) into a native result.excalidraw artifact named
// {format:"excalidraw"}, whose file bytes equal the submitted scene.
func TestCanvasSubmitWritesExcalidrawArtifact(t *testing.T) {
	srv, base := startTestServer(t, FeedbackRequest{ID: "s1", Surface: "canvas"})

	scene := []byte(`{"type":"excalidraw","elements":[{"id":"a","type":"rectangle"}],"appState":{}}`)
	putReq, _ := http.NewRequest(http.MethodPut, base+"/session/s1/draft", bytes.NewReader(scene))
	if _, err := http.DefaultClient.Do(putReq); err != nil {
		t.Fatalf("PUT draft: %v", err)
	}

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

	// property [7a]: envelope names {format:"excalidraw", path:"result.excalidraw"}.
	var found *Artifact
	for i := range result.Artifacts {
		if result.Artifacts[i].Format == "excalidraw" {
			found = &result.Artifacts[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("result artifacts %+v: no excalidraw artifact", result.Artifacts)
	}
	if found.Path != "result.excalidraw" {
		t.Fatalf("excalidraw artifact path = %q, want %q", found.Path, "result.excalidraw")
	}

	// property [7b]: the named file's bytes equal the submitted scene.
	got, err := os.ReadFile(filepath.Join(srv.dir, found.Path))
	if err != nil {
		t.Fatalf("read %s: %v", found.Path, err)
	}
	if !bytes.Equal(got, scene) {
		t.Fatalf("result.excalidraw = %s, want submitted scene %s", got, scene)
	}
}
