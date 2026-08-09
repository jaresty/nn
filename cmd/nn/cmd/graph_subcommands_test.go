package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// ── nn graph top ─────────────────────────────────────────────────────────────

func TestGraphTopPlain(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	hub := newTestNoteForCLI(note.GenerateID(), "Hub Note", note.TypeConcept)
	leaf1 := newTestNoteForCLI(note.GenerateID(), "Leaf One", note.TypeConcept)
	leaf2 := newTestNoteForCLI(note.GenerateID(), "Leaf Two", note.TypeConcept)
	leaf1.Links = []note.Link{{TargetID: hub.ID, Annotation: "points to hub"}}
	leaf2.Links = []note.Link{{TargetID: hub.ID, Annotation: "also points to hub"}}
	writeNoteFile(t, nbDir, hub)
	writeNoteFile(t, nbDir, leaf1)
	writeNoteFile(t, nbDir, leaf2)

	out, err := execute("graph", "top")
	if err != nil {
		t.Fatalf("nn graph top: %v", err)
	}
	if !strings.Contains(out, hub.ID) {
		t.Errorf("graph top: hub note %s not in output:\n%s", hub.ID, out)
	}
	if !strings.Contains(out, "Hub Note") {
		t.Errorf("graph top: hub title not in output:\n%s", out)
	}
}

func TestGraphTopLimit(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// Create 5 notes each receiving different inbound counts.
	notes := make([]*note.Note, 5)
	for i := range notes {
		n := newTestNoteForCLI(note.GenerateID(), "Note", note.TypeConcept)
		notes[i] = n
		writeNoteFile(t, nbDir, n)
	}
	// notes[0] gets 4 inbound, notes[1] gets 3, etc.
	for i := 1; i < 5; i++ {
		for j := 0; j < 5-i; j++ {
			src := newTestNoteForCLI(note.GenerateID(), "Src", note.TypeConcept)
			src.Links = []note.Link{{TargetID: notes[i-1].ID, Annotation: "link"}}
			writeNoteFile(t, nbDir, src)
		}
	}

	out, err := execute("graph", "top", "--limit", "2")
	if err != nil {
		t.Fatalf("nn graph top --limit 2: %v", err)
	}
	lines := nonEmptyLines(out)
	if len(lines) > 2 {
		t.Errorf("graph top --limit 2: got %d lines, want ≤2:\n%s", len(lines), out)
	}
}

func TestGraphTopJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	hub := newTestNoteForCLI(note.GenerateID(), "Hub", note.TypeConcept)
	src := newTestNoteForCLI(note.GenerateID(), "Src", note.TypeConcept)
	src.Links = []note.Link{{TargetID: hub.ID, Annotation: "link"}}
	writeNoteFile(t, nbDir, hub)
	writeNoteFile(t, nbDir, src)

	out, err := execute("graph", "top", "--format", "json")
	if err != nil {
		t.Fatalf("nn graph top --format json: %v", err)
	}
	var result []struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		InboundCount int    `json:"inbound_count"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("graph top --format json: invalid JSON: %v\n%s", err, out)
	}
	if len(result) == 0 {
		t.Fatal("graph top --format json: empty result")
	}
	if result[0].ID != hub.ID {
		t.Errorf("graph top --format json: first entry ID = %q, want %q", result[0].ID, hub.ID)
	}
	if result[0].InboundCount < 1 {
		t.Errorf("graph top --format json: inbound_count = %d, want ≥1", result[0].InboundCount)
	}
}

// ── nn graph orphans ──────────────────────────────────────────────────────────

func TestGraphOrphans(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	orphan := newTestNoteForCLI(note.GenerateID(), "Orphan Note", note.TypeConcept)
	connected := newTestNoteForCLI(note.GenerateID(), "Connected", note.TypeConcept)
	other := newTestNoteForCLI(note.GenerateID(), "Other", note.TypeConcept)
	globalProto := newTestNoteForCLI(note.GenerateID(), "Global Protocol", note.TypeProtocol)
	connected.Links = []note.Link{{TargetID: other.ID, Annotation: "link"}}
	writeNoteFile(t, nbDir, orphan)
	writeNoteFile(t, nbDir, connected)
	writeNoteFile(t, nbDir, other)
	writeNoteFile(t, nbDir, globalProto)

	out, err := execute("graph", "orphans")
	if err != nil {
		t.Fatalf("nn graph orphans: %v", err)
	}
	if !strings.Contains(out, orphan.ID) {
		t.Errorf("graph orphans: orphan %s not in output:\n%s", orphan.ID, out)
	}
	if strings.Contains(out, connected.ID) {
		t.Errorf("graph orphans: connected note %s should not appear:\n%s", connected.ID, out)
	}
	if strings.Contains(out, globalProto.ID) {
		t.Errorf("graph orphans: global protocol %s should not appear:\n%s", globalProto.ID, out)
	}
}

func TestGraphOrphansJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	orphan := newTestNoteForCLI(note.GenerateID(), "Orphan", note.TypeConcept)
	writeNoteFile(t, nbDir, orphan)

	out, err := execute("graph", "orphans", "--format", "json")
	if err != nil {
		t.Fatalf("nn graph orphans --format json: %v", err)
	}
	var result []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("graph orphans --format json: invalid JSON: %v\n%s", err, out)
	}
	found := false
	for _, r := range result {
		if r.ID == orphan.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("graph orphans --format json: orphan %s not in result", orphan.ID)
	}
}

// ── nn graph bridges ──────────────────────────────────────────────────────────

func TestGraphBridges(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// A → bridge → B: bridge connects two otherwise-unconnected notes.
	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	bridge := newTestNoteForCLI(note.GenerateID(), "Bridge Note", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "B", note.TypeConcept)
	a.Links = []note.Link{{TargetID: bridge.ID, Annotation: "to bridge"}}
	bridge.Links = []note.Link{{TargetID: b.ID, Annotation: "to b"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, bridge)
	writeNoteFile(t, nbDir, b)

	out, err := execute("graph", "bridges")
	if err != nil {
		t.Fatalf("nn graph bridges: %v", err)
	}
	if !strings.Contains(out, bridge.ID) {
		t.Errorf("graph bridges: bridge note %s not in output:\n%s", bridge.ID, out)
	}
}

func TestGraphBridgesJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	bridge := newTestNoteForCLI(note.GenerateID(), "Bridge", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "B", note.TypeConcept)
	a.Links = []note.Link{{TargetID: bridge.ID, Annotation: "link"}}
	bridge.Links = []note.Link{{TargetID: b.ID, Annotation: "link"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, bridge)
	writeNoteFile(t, nbDir, b)

	out, err := execute("graph", "bridges", "--format", "json")
	if err != nil {
		t.Fatalf("nn graph bridges --format json: %v", err)
	}
	var result []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Score int    `json:"score"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("graph bridges --format json: invalid JSON: %v\n%s", err, out)
	}
	if len(result) == 0 {
		t.Fatal("graph bridges --format json: empty result")
	}
	if result[0].ID != bridge.ID {
		t.Errorf("graph bridges --format json: first entry = %q, want %q", result[0].ID, bridge.ID)
	}
}

// ── nn graph show ─────────────────────────────────────────────────────────────

func TestGraphShowFocusJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	center := newTestNoteForCLI(note.GenerateID(), "Center", note.TypeConcept)
	neighbor := newTestNoteForCLI(note.GenerateID(), "Neighbor", note.TypeConcept)
	center.Links = []note.Link{{TargetID: neighbor.ID, Annotation: "link"}}
	writeNoteFile(t, nbDir, center)
	writeNoteFile(t, nbDir, neighbor)

	out, err := execute("graph", "show", "--focus", center.ID, "--format", "json")
	if err != nil {
		t.Fatalf("nn graph show --focus --format json: %v", err)
	}
	var result struct {
		Center string `json:"center"`
		Nodes  []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"nodes"`
		Edges []struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"edges"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("graph show --format json: invalid JSON: %v\n%s", err, out)
	}
	if result.Center != center.ID {
		t.Errorf("graph show: center = %q, want %q", result.Center, center.ID)
	}
	if len(result.Nodes) < 2 {
		t.Errorf("graph show: got %d nodes, want ≥2", len(result.Nodes))
	}
	if len(result.Edges) < 1 {
		t.Errorf("graph show: got %d edges, want ≥1", len(result.Edges))
	}
}

func TestGraphShowFullJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "B", note.TypeConcept)
	a.Links = []note.Link{{TargetID: b.ID, Annotation: "link"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	out, err := execute("graph", "show", "--format", "json")
	if err != nil {
		t.Fatalf("nn graph show --format json (no focus): %v", err)
	}
	var result struct {
		Nodes []struct{ ID string `json:"id"` } `json:"nodes"`
		Edges []struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"edges"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("graph show full --format json: invalid JSON: %v\n%s", err, out)
	}
	if len(result.Nodes) < 2 {
		t.Errorf("graph show full: got %d nodes, want ≥2", len(result.Nodes))
	}
	if len(result.Edges) < 1 {
		t.Errorf("graph show full: got %d edges, want ≥1", len(result.Edges))
	}
}

// ── nn graph export --format dot ─────────────────────────────────────────────

func TestGraphExportDOT(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "B", note.TypeConcept)
	a.Links = []note.Link{{TargetID: b.ID, Annotation: "link"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	out, err := execute("graph", "export", "--format", "dot")
	if err != nil {
		t.Fatalf("nn graph export --format dot: %v", err)
	}
	if !strings.Contains(out, "digraph") {
		t.Errorf("graph export --format dot: missing 'digraph':\n%s", out)
	}
	if !strings.Contains(out, "->") {
		t.Errorf("graph export --format dot: missing '->' edge:\n%s", out)
	}
}

// ── nn graph export --format html ────────────────────────────────────────────

func TestGraphExportHTMLFiltersBrokenLinks(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	// Link to a non-existent target.
	a.Links = []note.Link{{TargetID: "nonexistent-id-999", Annotation: "broken"}}
	writeNoteFile(t, nbDir, a)

	out, err := execute("graph", "export", "--format", "html")
	if err != nil {
		t.Fatalf("nn graph export --format html: %v", err)
	}
	if strings.Contains(out, `"nonexistent-id-999"`) {
		t.Errorf("html export: broken-link target should be filtered from edges JSON")
	}
}

func TestGraphExportHTMLEmbedsBodies(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	a.Body = "This is the note body content."
	writeNoteFile(t, nbDir, a)

	out, err := execute("graph", "export", "--format", "html")
	if err != nil {
		t.Fatalf("nn graph export --format html: %v", err)
	}
	if !strings.Contains(out, `"body"`) {
		t.Errorf("html export: node JSON missing 'body' key")
	}
	if !strings.Contains(out, "This is the note body content.") {
		t.Errorf("html export: note body content not found in output")
	}
}

func TestGraphExportHTMLSidePanel(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	writeNoteFile(t, nbDir, a)

	out, err := execute("graph", "export", "--format", "html")
	if err != nil {
		t.Fatalf("nn graph export --format html: %v", err)
	}
	if !strings.Contains(out, "panel") {
		t.Errorf("html export: missing side panel element")
	}
}

func TestGraphExportHTML(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "Beta", note.TypeConcept)
	a.Links = []note.Link{{TargetID: b.ID, Annotation: "link"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	out, err := execute("graph", "export", "--format", "html")
	if err != nil {
		t.Fatalf("nn graph export --format html: %v", err)
	}
	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Errorf("html export: missing DOCTYPE:\n%.200s", out)
	}
	if !strings.Contains(out, "<script") {
		t.Errorf("html export: missing <script>:\n%.200s", out)
	}
	if !strings.Contains(out, `"nodes"`) {
		t.Errorf("html export: missing graph nodes JSON:\n%.200s", out)
	}
	if !strings.Contains(out, "highlight") {
		t.Errorf("html export: missing highlight interaction:\n%.200s", out)
	}
	if !strings.Contains(out, "const graph =") {
		t.Errorf("html export: missing static graph embed (const graph =):\n%.200s", out)
	}
}

func TestGraphExportHTMLStaticEmbeddedData(t *testing.T) {
	// Static export: the HTML must be self-contained — graph data is embedded and parseable,
	// not fetched at runtime.
	nbDir, execute := setupNotebook(t)

	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "Beta", note.TypeConcept)
	a.Links = []note.Link{{TargetID: b.ID, Annotation: "link"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	out, err := execute("graph", "export", "--format", "html")
	if err != nil {
		t.Fatalf("static export: %v", err)
	}

	// Extract the JSON object assigned to `const graph = {...};`
	const marker = "const graph = "
	idx := strings.Index(out, marker)
	if idx == -1 {
		t.Fatalf("static export: graph data not embedded (missing %q)", marker)
	}
	jsonStart := idx + len(marker)
	// Find the matching closing brace (the JSON ends at `;`)
	semi := strings.Index(out[jsonStart:], ";\n")
	if semi == -1 {
		t.Fatalf("static export: could not find end of embedded graph JSON")
	}
	jsonStr := out[jsonStart : jsonStart+semi]

	var graph struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
		Edges []struct {
			Source string `json:"source"`
			Target string `json:"target"`
		} `json:"edges"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &graph); err != nil {
		t.Fatalf("static export: embedded graph JSON not parseable: %v\n%.200s", err, jsonStr)
	}

	ids := make(map[string]bool)
	for _, n := range graph.Nodes {
		ids[n.ID] = true
	}
	if !ids[a.ID] {
		t.Errorf("static export: node %s (Alpha) missing from embedded graph", a.ID)
	}
	if !ids[b.ID] {
		t.Errorf("static export: node %s (Beta) missing from embedded graph", b.ID)
	}
	if len(graph.Edges) == 0 {
		t.Errorf("static export: no edges in embedded graph")
	}
}

func TestGraphExportHTMLServeModeNoEmbeddedData(t *testing.T) {
	// Serve mode: the HTML must NOT embed graph data — it fetches from /graph at runtime.
	// Verify by calling buildHTML(serveMode=true) and asserting that the embedded-data
	// marker is absent, meaning the browser must fetch graph data rather than reading
	// it from a static variable.
	html, err := buildHTML(nil, nil, nil, true)
	if err != nil {
		t.Fatalf("buildHTML serveMode=true: %v", err)
	}
	out := string(html)
	if strings.Contains(out, "const graph = ") {
		t.Errorf("serve mode: graph data must not be embedded (found 'const graph = ')")
	}
}

func TestGraphExportHTMLLayoutToggle(t *testing.T) {
	// property [1a]: a layout toggle element must be present in the DOM
	nbDir, execute := setupNotebook(t)
	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	writeNoteFile(t, nbDir, a)

	out, err := execute("graph", "export", "--format", "html")
	if err != nil {
		t.Fatalf("html export: %v", err)
	}
	if !strings.Contains(out, `id="btn-layout"`) {
		t.Errorf("html export: layout toggle button (id=btn-layout) missing from output")
	}
}


// ── search/click interaction [P1a P1b P2a P2b] ───────────────────────────────

func TestGraphExportHTMLSearchClearsHighlight(t *testing.T) {
	// [P1a] search handler must call clearHighlight() before applying search-dim
	html, err := buildHTML(nil, nil, nil, true)
	if err != nil {
		t.Fatalf("buildHTML: %v", err)
	}
	out := string(html)
	if !strings.Contains(out, `clearHighlight()`) {
		t.Errorf("[P1a] search handler must call clearHighlight() before applying search-dim")
	}
}

func TestGraphExportHTMLClickClearsSearch(t *testing.T) {
	// [P1b] node click handler must clear the search input and search-dim on click
	html, err := buildHTML(nil, nil, nil, true)
	if err != nil {
		t.Fatalf("buildHTML: %v", err)
	}
	out := string(html)
	if !strings.Contains(out, `si.value = ""`) {
		t.Errorf("[P1b] node click handler must clear search input (si.value = \"\")")
	}
	if !strings.Contains(out, `classed("search-dim", false)`) {
		t.Errorf("[P1b] node click handler must clear search-dim class")
	}
}

// ── node size by degree [P3a P3b] ────────────────────────────────────────────

func TestGraphExportHTMLNodeSizeByDegree(t *testing.T) {
	// [P3a] node radius must scale with degree — d3.scaleSqrt or similar degree-based scale
	html, err := buildHTML(nil, nil, nil, false)
	if err != nil {
		t.Fatalf("buildHTML: %v", err)
	}
	out := string(html)
	if !strings.Contains(out, `scaleSqrt`) && !strings.Contains(out, `scaleLinear`) {
		t.Errorf("[P3a] node radius must use a D3 scale (scaleSqrt/scaleLinear) based on degree")
	}
}

func TestGraphExportHTMLNodeSizeMinRadius(t *testing.T) {
	// [P3b] degree-0 nodes must have a non-zero minimum radius in the scale range
	html, err := buildHTML(nil, nil, nil, false)
	if err != nil {
		t.Fatalf("buildHTML: %v", err)
	}
	out := string(html)
	// range([minRadius, maxRadius]) — minRadius must be > 0
	if !strings.Contains(out, `.range([`) {
		t.Errorf("[P3b] radius scale must have an explicit range with non-zero minimum")
	}
}

// ── status overlay [P4a P4b P4c] ─────────────────────────────────────────────

func TestGraphExportHTMLStatusTogglePresent(t *testing.T) {
	// [P4a] status toggle button must be present in the HTML
	html, err := buildHTML(nil, nil, nil, false)
	if err != nil {
		t.Fatalf("buildHTML: %v", err)
	}
	out := string(html)
	if !strings.Contains(out, `id="btn-status"`) {
		t.Errorf("[P4a] status toggle button (id=btn-status) missing from HTML")
	}
}

func TestGraphExportHTMLStatusToggleCycles(t *testing.T) {
	// [P4a] status cycle labels must appear: All, Draft, Reviewed, Permanent
	html, err := buildHTML(nil, nil, nil, false)
	if err != nil {
		t.Fatalf("buildHTML: %v", err)
	}
	out := string(html)
	for _, label := range []string{"All", "Draft", "Reviewed", "Permanent"} {
		if !strings.Contains(out, label) {
			t.Errorf("[P4a] status cycle label %q missing from HTML", label)
		}
	}
}

func TestGraphServeStatusFieldInGraph(t *testing.T) {
	// [P4b] /graph JSON must include 'status' field on each node
	nbDir, cfgFile := setupNotebookWithCfg(t)
	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	writeNoteFile(t, nbDir, a)

	port := 17350
	startServeForTest(t, port, cfgFile)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/graph", port))
	if err != nil {
		t.Fatalf("GET /graph: %v", err)
	}
	defer resp.Body.Close()

	var body struct {
		Nodes []map[string]any `json:"nodes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("GET /graph: body not valid JSON: %v", err)
	}
	if len(body.Nodes) == 0 {
		t.Fatal("GET /graph: no nodes")
	}
	if _, ok := body.Nodes[0]["status"]; !ok {
		t.Errorf("[P4b] /graph node JSON missing 'status' field")
	}
}

// ── selection tray + export [P5a-5e] ─────────────────────────────────────────

func TestGraphExportHTMLTrayPresent(t *testing.T) {
	// [P5d] tray count element and Export button must be in both static and serve HTML
	for _, serveMode := range []bool{false, true} {
		html, err := buildHTML(nil, nil, nil, serveMode)
		if err != nil {
			t.Fatalf("buildHTML serveMode=%v: %v", serveMode, err)
		}
		out := string(html)
		if !strings.Contains(out, `id="tray-count"`) {
			t.Errorf("[P5d] serveMode=%v: tray count element (id=tray-count) missing", serveMode)
		}
		if !strings.Contains(out, `id="btn-export"`) {
			t.Errorf("[P5d] serveMode=%v: Export button (id=btn-export) missing", serveMode)
		}
	}
}

func TestGraphExportHTMLTrayShiftClick(t *testing.T) {
	// [P5a P5b] shift-click handler must toggle node in/out of tray and update count
	html, err := buildHTML(nil, nil, nil, false)
	if err != nil {
		t.Fatalf("buildHTML: %v", err)
	}
	out := string(html)
	if !strings.Contains(out, `shiftKey`) {
		t.Errorf("[P5a/5b] shift-click handler (shiftKey) missing from HTML")
	}
	if !strings.Contains(out, `tray-count`) {
		t.Errorf("[P5a/5b] tray count update missing from shift-click handler")
	}
}

func TestGraphExportHTMLExportCopiesContent(t *testing.T) {
	// [P5c] Export button must call clipboard writeText with note title, body, links
	html, err := buildHTML(nil, nil, nil, false)
	if err != nil {
		t.Fatalf("buildHTML: %v", err)
	}
	out := string(html)
	if !strings.Contains(out, `clipboard.writeText`) {
		t.Errorf("[P5c] Export button must call navigator.clipboard.writeText")
	}
}

func TestGraphExportHTMLExportDisabledWhenEmpty(t *testing.T) {
	// [P5e] Export button must be disabled when tray is empty
	html, err := buildHTML(nil, nil, nil, false)
	if err != nil {
		t.Fatalf("buildHTML: %v", err)
	}
	out := string(html)
	if !strings.Contains(out, `btn-export`) || !strings.Contains(out, `disabled`) {
		t.Errorf("[P5e] Export button must be disabled when tray is empty")
	}
}

// ── remove chat [P16 P17] ─────────────────────────────────────────────────────

func TestGraphExportNoChatFlag(t *testing.T) {
	// [P16] --chat flag must not appear in --help output
	nbDir, execute := setupNotebook(t)
	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	writeNoteFile(t, nbDir, a)

	out, _ := execute("graph", "export", "--help")
	if strings.Contains(out, "--chat") {
		t.Errorf("[P16] --chat flag must not appear in graph export --help after removal")
	}
}

func TestGraphExportHTMLNoChatElements(t *testing.T) {
	// [P17] msg-panel, msg-input, msg-send must not appear in any generated HTML
	for _, chatMode := range []bool{false, true} {
		html, err := buildHTML(nil, nil, nil, true)
		if err != nil {
			t.Fatalf("buildHTML chatMode=%v: %v", chatMode, err)
		}
		out := string(html)
		for _, id := range []string{`id="msg-panel"`, `id="msg-input"`, `id="msg-send"`} {
			if strings.Contains(out, id) {
				t.Errorf("[P17] chatMode=%v: %s must not appear in HTML after chat removal", chatMode, id)
			}
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}
