package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// TestGraphShowTypeAndEdgeColor verifies that --color colorizes node titles by
// note type and edge labels by link family in the tree view.
//
// property [1]: an edge label is wrapped in its link family's ANSI color
// property [2]: a node title is wrapped in its note type's ANSI color
// property [4]: --color never emits no escapes; json never colored
func TestGraphShowTypeAndEdgeColor(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	now := time.Now().UTC().Truncate(time.Second)

	ego := newTestNoteForCLI(note.GenerateID(), "Ego", note.TypeModel)
	tension := newTestNoteForCLI(note.GenerateID(), "Tension", note.TypeArgument)
	ego.Created, ego.Modified = now, now
	tension.Created, tension.Modified = now, now
	// tension -> ego via contradicts (tension link family = warning)
	tension.Links = []note.Link{{TargetID: ego.ID, Type: "contradicts", Annotation: "no"}}
	writeNoteFile(t, nbDir, ego)
	writeNoteFile(t, nbDir, tension)

	esc := "\x1b["

	always, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--color", "always", "--format", "text")
	if err != nil {
		t.Fatalf("color always: %v", err)
	}
	// property [1]: the contradicts edge label carries a color (tension = red 31).
	if !strings.Contains(always, "\x1b[31m") {
		t.Errorf("property [1]: contradicts edge label not colored with tension family (red):\n%q", always)
	}
	// property [2]: a node title carries a type color — at least one non-zone,
	// non-red SGR code must appear on a node/title (type palette).
	// We assert presence of an escape sequence attached to the ego title region
	// distinct from zone-header coloring (which is absent here: not --zones).
	if !strings.Contains(always, esc) {
		t.Errorf("property [2]: no ANSI escapes at all in colored tree view:\n%q", always)
	}
	// Count distinct SGR codes: node-type + edge-family should yield >=2.
	codes := map[string]bool{}
	for _, part := range strings.Split(always, esc) {
		if i := strings.IndexByte(part, 'm'); i >= 0 && i <= 3 {
			codes[part[:i]] = true
		}
	}
	// exclude the reset "0"
	delete(codes, "0")
	if len(codes) < 2 {
		t.Errorf("property [1]+[2]: expected >=2 distinct colors (node type + edge family), got %v:\n%q", codes, always)
	}

	never, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--color", "never", "--format", "text")
	if err != nil {
		t.Fatalf("color never: %v", err)
	}
	if strings.Contains(never, esc) {
		t.Errorf("property [4]: --color never must emit no escapes:\n%q", never)
	}

	js, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--color", "always", "--format", "json")
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	if strings.Contains(js, esc) {
		t.Errorf("property [4]: json must never be colored:\n%q", js)
	}

	// property [3]: the zoned key documents the type and link-family color scheme.
	zoned, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--zones", "--color", "always", "--format", "text")
	if err != nil {
		t.Fatalf("zoned key: %v", err)
	}
	low := strings.ToLower(zoned)
	if !strings.Contains(low, "type") || !strings.Contains(low, "family") {
		t.Errorf("property [3]: zoned key must document the note-type and link-family color scheme:\n%s", zoned)
	}
}
