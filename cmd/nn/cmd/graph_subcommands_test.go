package cmd

import (
	"encoding/json"
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
