package feedback

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// property [4a]+[4b]: submitting the graph surface materializes a graph-native
// result artifact (format=graph-selection) whose content carries the selected
// node ids, per-node annotations, and the free-text answer, and the thin
// envelope names that artifact by path.
func TestGraphSubmitWritesGraphSelectionArtifact(t *testing.T) {
	q := FeedbackRequest{ID: "s1", Surface: "graph", Focus: "ego", AllowedNodes: []string{"ego", "nbr"}}
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
	base := "http://" + srv.Addr()

	draft := []byte(`{"selected":["ego","nbr"],"annotations":{"ego":"core claim","nbr":"supports it"},"answer":"nbr is the strongest support"}`)
	putReq, _ := http.NewRequest(http.MethodPut, base+"/session/s1/draft", bytes.NewReader(draft))
	if _, err := http.DefaultClient.Do(putReq); err != nil {
		t.Fatalf("PUT draft: %v", err)
	}

	go func() { srv.Wait() }()
	resp, err := http.Post(base+"/session/s1/submit", "application/json", nil)
	if err != nil {
		t.Fatalf("POST submit: %v", err)
	}
	resp.Body.Close()

	result, err := ReadResult(dir)
	if err != nil {
		t.Fatalf("ReadResult: %v", err)
	}

	// property [4a]: envelope names a graph-selection artifact.
	var art *Artifact
	for i := range result.Artifacts {
		if result.Artifacts[i].Format == "graph-selection" {
			art = &result.Artifacts[i]
		}
	}
	if art == nil {
		t.Fatalf("no graph-selection artifact in envelope: %+v", result.Artifacts)
	}

	// property [4b]: the artifact content carries ids, annotations, and answer.
	body, err := os.ReadFile(filepath.Join(dir, art.Path))
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	var payload struct {
		Selected    []string          `json:"selected"`
		Annotations map[string]string `json:"annotations"`
		Answer      string            `json:"answer"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("artifact not valid graph-selection JSON: %v\n%s", err, body)
	}
	if strings.Join(payload.Selected, ",") != "ego,nbr" {
		t.Fatalf("selected = %v, want [ego nbr]", payload.Selected)
	}
	if payload.Annotations["ego"] != "core claim" {
		t.Fatalf("annotations not carried: %v", payload.Annotations)
	}
	if payload.Answer != "nbr is the strongest support" {
		t.Fatalf("answer not carried: %q", payload.Answer)
	}
}
