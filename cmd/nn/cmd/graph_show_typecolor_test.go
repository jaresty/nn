package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// TestGraphShowTypeAndEdgeMarkers verifies that --color emits colored-circle
// emoji markers (which survive markdown/relay) instead of ANSI escapes: node
// titles prefixed by a note-type circle, edge labels by a link-family circle.
//
// property [1]:  node title prefixed by its note-type emoji
// property [2]:  edge label prefixed by its link-family emoji
// property [4a]: --color never emits no markers; json never marked
// property [4b]: no raw ANSI escape appears in any text output
func TestGraphShowTypeAndEdgeMarkers(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	now := time.Now().UTC().Truncate(time.Second)

	ego := newTestNoteForCLI(note.GenerateID(), "Ego", note.TypeModel)
	tension := newTestNoteForCLI(note.GenerateID(), "Tension", note.TypeArgument)
	ego.Created, ego.Modified = now, now
	tension.Created, tension.Modified = now, now
	tension.Links = []note.Link{{TargetID: ego.ID, Type: "contradicts", Annotation: "no"}}
	writeNoteFile(t, nbDir, ego)
	writeNoteFile(t, nbDir, tension)

	esc := "\x1b"
	const (
		modelEmoji   = "🟣" // model
		tensionEmoji = "🔴" // tension family (contradicts/questions) + argument type
	)

	always, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--color", "always", "--format", "text")
	if err != nil {
		t.Fatalf("color always: %v", err)
	}
	// property [4b]: no raw ANSI escapes anywhere.
	if strings.Contains(always, esc) {
		t.Errorf("property [4b]: text output must contain no raw ANSI escape:\n%q", always)
	}
	// property [1]: the model ego node title carries the model emoji.
	if !strings.Contains(always, modelEmoji) {
		t.Errorf("property [1]: model node not prefixed with %s:\n%q", modelEmoji, always)
	}
	// property [2]: the contradicts edge carries the tension emoji.
	if !strings.Contains(always, tensionEmoji) {
		t.Errorf("property [2]: tension edge not prefixed with %s:\n%q", tensionEmoji, always)
	}

	// property [4a]: --color never suppresses all markers.
	never, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--color", "never", "--format", "text")
	if err != nil {
		t.Fatalf("color never: %v", err)
	}
	if strings.Contains(never, modelEmoji) || strings.Contains(never, tensionEmoji) {
		t.Errorf("property [4a]: --color never must emit no emoji markers:\n%q", never)
	}
	if strings.Contains(never, esc) {
		t.Errorf("property [4b]: --color never must emit no ANSI either:\n%q", never)
	}

	// property [4a]: json never marked (and never ANSI).
	js, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--color", "always", "--format", "json")
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	if strings.Contains(js, modelEmoji) || strings.Contains(js, esc) {
		t.Errorf("property [4a]/[4b]: json must have neither emoji markers nor ANSI:\n%q", js)
	}

	// property [3]: zoned key documents the emoji legends.
	zoned, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--zones", "--color", "always", "--format", "text")
	if err != nil {
		t.Fatalf("zoned: %v", err)
	}
	low := strings.ToLower(zoned)
	if !strings.Contains(low, "type") || !strings.Contains(low, "family") || !strings.Contains(zoned, modelEmoji) {
		t.Errorf("property [3]: zoned key must show type and family emoji legends:\n%s", zoned)
	}
}
