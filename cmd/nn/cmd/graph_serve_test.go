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

// property F1/F2: /graph?focus=E includes notes that link TO E (incoming),
// not just outgoing neighbors, and each such node carries its inbound zone —
// so the viewer's Zoned mode shows the full four-zone spread.
func TestGraphServeGraphFocusIncludesIncoming(t *testing.T) {
	nbDir, cfgFile := setupNotebookWithCfg(t)

	ego := newTestNoteForCLI(note.GenerateID(), "Ego", note.TypeModel)
	foe := newTestNoteForCLI(note.GenerateID(), "Foe", note.TypeArgument)
	// foe -> ego via contradicts (incoming to ego => left zone)
	foe.Links = []note.Link{{TargetID: ego.ID, Type: "contradicts", Annotation: "opposes"}}
	writeNoteFile(t, nbDir, ego)
	writeNoteFile(t, nbDir, foe)

	port := 17348
	startServeForTest(t, port, cfgFile)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/graph?focus=%s&depth=1", port, ego.ID))
	if err != nil {
		t.Fatalf("GET /graph?focus: %v", err)
	}
	defer resp.Body.Close()

	var body struct {
		Nodes []map[string]any `json:"nodes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("GET /graph?focus: body not valid JSON: %v", err)
	}

	var foeZone any
	var foeFound bool
	for _, n := range body.Nodes {
		if n["id"] == foe.ID {
			foeFound = true
			foeZone = n["zone"]
		}
	}
	// property F1: the incoming neighbor is present in the ego subgraph.
	if !foeFound {
		t.Fatalf("property F1: incoming neighbor Foe (%s) missing from /graph?focus node set", foe.ID)
	}
	// property F2: it carries its inbound zone (contradicts in => left).
	if foeZone != "left" {
		t.Errorf("property F2: Foe zone = %v, want %q", foeZone, "left")
	}
}

// property S2: /graph?focus=E annotates each node with its directional zone
// (via zoneOf on the node's direct edge to E), so the viewer's Zoned layout
// reads server-computed zones rather than recomputing the mapping in JS.
func TestGraphServeGraphFocusZone(t *testing.T) {
	nbDir, cfgFile := setupNotebookWithCfg(t)

	ego := newTestNoteForCLI(note.GenerateID(), "Ego", note.TypeModel)
	up := newTestNoteForCLI(note.GenerateID(), "Up", note.TypeConcept)
	ego.Links = []note.Link{{TargetID: up.ID, Type: "extends", Annotation: "builds on"}}
	writeNoteFile(t, nbDir, ego)
	writeNoteFile(t, nbDir, up)

	port := 17347
	startServeForTest(t, port, cfgFile)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/graph?focus=%s&depth=1", port, ego.ID))
	if err != nil {
		t.Fatalf("GET /graph?focus: %v", err)
	}
	defer resp.Body.Close()

	var body struct {
		Nodes []map[string]any `json:"nodes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("GET /graph?focus: body not valid JSON: %v", err)
	}

	zoneByID := map[string]any{}
	for _, n := range body.Nodes {
		zoneByID[n["id"].(string)] = n["zone"]
	}
	// ego -> up via extends (out) => top
	if zoneByID[up.ID] != "top" {
		t.Errorf("property S2: node Up zone = %v, want %q", zoneByID[up.ID], "top")
	}
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

// property P1: each node in a /graph?focus response carries its full-graph
// degree (in+out incident edges), so the viewer can show how many direct
// connections are hidden by the depth=1 focus.
func TestGraphServeGraphFocusDegree(t *testing.T) {
	nbDir, cfgFile := setupNotebookWithCfg(t)

	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "Beta", note.TypeConcept)
	c := newTestNoteForCLI(note.GenerateID(), "Gamma", note.TypeConcept)
	// a -> b -> c. Focus a depth=1 returns a,b. b's full-graph degree is 2
	// (a->b and b->c), of which only a->b is visible, so 1 is hidden.
	a.Links = []note.Link{{TargetID: b.ID, Annotation: "link"}}
	b.Links = []note.Link{{TargetID: c.ID, Annotation: "link"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)
	writeNoteFile(t, nbDir, c)

	port := 17352
	startServeForTest(t, port, cfgFile)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/graph?focus=%s&depth=1", port, a.ID))
	if err != nil {
		t.Fatalf("GET /graph?focus: %v", err)
	}
	defer resp.Body.Close()

	var body struct {
		Nodes []map[string]any `json:"nodes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("GET /graph?focus: body not valid JSON: %v", err)
	}
	degByID := map[string]float64{}
	for _, n := range body.Nodes {
		if d, ok := n["degree"].(float64); ok {
			degByID[n["id"].(string)] = d
		} else {
			t.Fatalf("property P1: node %v missing numeric degree field", n["id"])
		}
	}
	if degByID[b.ID] != 2 {
		t.Errorf("property P1: node b degree = %v, want 2 (a->b, b->c)", degByID[b.ID])
	}
	if degByID[a.ID] != 1 {
		t.Errorf("property P1: node a degree = %v, want 1 (a->b)", degByID[a.ID])
	}
}


// property S1: every edge in the /graph response carries the source note's
// link type in a "type" field (so the viewer can compute directional zones).
func TestGraphServeGraphEdgeType(t *testing.T) {
	nbDir, cfgFile := setupNotebookWithCfg(t)

	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "Beta", note.TypeConcept)
	a.Links = []note.Link{{TargetID: b.ID, Type: "contradicts", Annotation: "opposes"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	port := 17346
	startServeForTest(t, port, cfgFile)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/graph", port))
	if err != nil {
		t.Fatalf("GET /graph: %v", err)
	}
	defer resp.Body.Close()

	var body struct {
		Edges []map[string]any `json:"edges"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("GET /graph: body not valid JSON: %v", err)
	}

	var found bool
	for _, e := range body.Edges {
		if e["source"] == a.ID && e["target"] == b.ID {
			found = true
			if e["type"] != "contradicts" {
				t.Errorf("property S1: edge a->b type = %v, want %q", e["type"], "contradicts")
			}
		}
	}
	if !found {
		t.Fatalf("property S1: expected an edge from %s to %s in /graph response", a.ID, b.ID)
	}
}

func TestGraphServePostEvent(t *testing.T) {
	// property [3]: POST /event with {id, title} must succeed (204) — event is logged to stdout
	nbDir, cfgFile := setupNotebookWithCfg(t)
	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	writeNoteFile(t, nbDir, a)

	port := 17345
	startServeForTest(t, port, cfgFile)

	body := strings.NewReader(`{"id":"` + a.ID + `","title":"Alpha"}`)
	resp, err := http.Post(fmt.Sprintf("http://localhost:%d/event", port), "application/json", body)
	if err != nil {
		t.Fatalf("POST /event: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("POST /event: status = %d, want 204", resp.StatusCode)
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

func TestGraphServeChatEndpointRemoved(t *testing.T) {
	// [P18] POST /chat must return 404 or 405 after chat removal
	nbDir, cfgFile := setupNotebookWithCfg(t)
	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	writeNoteFile(t, nbDir, a)

	port := 17347
	startServeForTest(t, port, cfgFile)

	resp, err := http.Post(fmt.Sprintf("http://localhost:%d/chat", port), "application/json", strings.NewReader(`{"text":"hello"}`))
	if err != nil {
		t.Fatalf("POST /chat: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("[P18] POST /chat: status = %d, want 404 or 405 after removal", resp.StatusCode)
	}
}

func TestGraphServeMessagesEndpointRemoved(t *testing.T) {
	// [P19] GET /messages must return 404 or 405 after chat removal
	nbDir, cfgFile := setupNotebookWithCfg(t)
	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	writeNoteFile(t, nbDir, a)

	port := 17348
	startServeForTest(t, port, cfgFile)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/messages", port))
	if err != nil {
		t.Fatalf("GET /messages: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("[P19] GET /messages: status = %d, want 404 or 405 after removal", resp.StatusCode)
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
