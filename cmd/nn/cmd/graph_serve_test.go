package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// startServeForTest launches graph export --format html --serve --port <port>
// in a background goroutine using a cancellable context.
// Registers t.Cleanup to cancel and wait for the server to exit.
func startServeForTest(t *testing.T, port int, cfgFile string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		root := NewRootCmdForTest(cfgFile)
		root.SetArgs([]string{
			"graph", "export", "--format", "html",
			"--serve", "--port", fmt.Sprintf("%d", port),
		})
		root.SetContext(ctx)
		_ = root.Execute()
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	})
	// Poll until port accepts connections or timeout.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	t.Fatalf("serve: server did not become ready on port %d within 3s", port)
}

func TestGraphServeGetGraphFocus(t *testing.T) {
	nbDir, cfgFile := setupNotebookWithCfg(t)

	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "Beta", note.TypeConcept)
	c := newTestNoteForCLI(note.GenerateID(), "Gamma", note.TypeConcept)
	// a -> b -> c; focus on a with depth=1 should return only a and b, not c
	a.Links = []note.Link{{TargetID: b.ID, Annotation: "link"}}
	b.Links = []note.Link{{TargetID: c.ID, Annotation: "link"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)
	writeNoteFile(t, nbDir, c)

	port := 17344
	startServeForTest(t, port, cfgFile)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/graph?focus=%s&depth=1", port, a.ID))
	if err != nil {
		t.Fatalf("GET /graph?focus: %v", err)
	}
	defer resp.Body.Close()

	var body struct {
		Nodes []map[string]any `json:"nodes"`
		Edges []map[string]any `json:"edges"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("GET /graph?focus: body not valid JSON: %v", err)
	}
	if len(body.Nodes) != 2 {
		t.Errorf("GET /graph?focus depth=1: expected 2 nodes (a,b), got %d", len(body.Nodes))
	}
	for _, n := range body.Nodes {
		if n["id"] == c.ID {
			t.Errorf("GET /graph?focus depth=1: node c should not appear at depth=1")
		}
	}
}

func TestGraphServeGetRoot(t *testing.T) {
	nbDir, cfgFile := setupNotebookWithCfg(t)

	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	writeNoteFile(t, nbDir, a)

	port := 17341
	startServeForTest(t, port, cfgFile)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Errorf("GET /: Content-Type = %q, want text/html", ct)
	}
}

func TestGraphServePostSearch(t *testing.T) {
	nbDir, cfgFile := setupNotebookWithCfg(t)

	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	a.Body = "authentication security token"
	writeNoteFile(t, nbDir, a)

	port := 17343
	startServeForTest(t, port, cfgFile)

	body := strings.NewReader(`{"query":"authentication"}`)
	resp, err := http.Post(fmt.Sprintf("http://localhost:%d/search", port), "application/json", body)
	if err != nil {
		t.Fatalf("POST /search: %v", err)
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("POST /search: Content-Type = %q, want application/json", ct)
	}
	var results []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatalf("POST /search: body is not a valid JSON array: %v", err)
	}
	for _, r := range results {
		if _, ok := r["id"]; !ok {
			t.Errorf("POST /search: result missing 'id' field: %v", r)
		}
		if _, ok := r["score"]; !ok {
			t.Errorf("POST /search: result missing 'score' field: %v", r)
		}
	}
}

func TestGraphServeGetGraph(t *testing.T) {
	nbDir, cfgFile := setupNotebookWithCfg(t)

	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	writeNoteFile(t, nbDir, a)

	port := 17342
	startServeForTest(t, port, cfgFile)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/graph", port))
	if err != nil {
		t.Fatalf("GET /graph: %v", err)
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("GET /graph: Content-Type = %q, want application/json", ct)
	}

	var body struct {
		Nodes []map[string]any `json:"nodes"`
		Edges []map[string]any `json:"edges"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("GET /graph: body is not valid JSON: %v", err)
	}
	if len(body.Nodes) != 1 {
		t.Errorf("GET /graph: expected 1 node, got %d", len(body.Nodes))
	}
}
