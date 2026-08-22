package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// TestGraphShowZonesLegend verifies the zoned text view includes a key that
// maps each zone to its link types.
//
// property [3]: zoned text output contains a zone -> link-type key.
func TestGraphShowZonesLegend(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	now := time.Now().UTC().Truncate(time.Second)
	ego := newTestNoteForCLI(note.GenerateID(), "Ego", note.TypeModel)
	up := newTestNoteForCLI(note.GenerateID(), "Up", note.TypeConcept)
	ego.Created, ego.Modified = now, now
	up.Created, up.Modified = now, now
	ego.Links = []note.Link{{TargetID: up.ID, Type: "extends", Annotation: "x"}}
	writeNoteFile(t, nbDir, ego)
	writeNoteFile(t, nbDir, up)

	out, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--zones", "--format", "text")
	if err != nil {
		t.Fatalf("graph show --zones: %v", err)
	}
	// The key must name a zone and at least one of its link types.
	if !strings.Contains(out, "TOP") || !strings.Contains(strings.ToLower(out), "extends") {
		t.Errorf("property [3]: zone->link-type key missing (expected a legend naming zones and link types):\n%s", out)
	}
	// The legend is a distinct labeled block, not just the zone group headers.
	if !strings.Contains(strings.ToLower(out), "key") && !strings.Contains(strings.ToLower(out), "legend") {
		t.Errorf("property [3]: no key/legend block found:\n%s", out)
	}
}

// TestGraphShowColor verifies --color gates ANSI escapes.
//
// property [4a]: color=never has no ESC; color=always has ESC (text)
// property [4b]: json output never contains ESC regardless of --color
func TestGraphShowColor(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	now := time.Now().UTC().Truncate(time.Second)
	ego := newTestNoteForCLI(note.GenerateID(), "Ego", note.TypeModel)
	up := newTestNoteForCLI(note.GenerateID(), "Up", note.TypeConcept)
	ego.Created, ego.Modified = now, now
	up.Created, up.Modified = now, now
	ego.Links = []note.Link{{TargetID: up.ID, Type: "extends", Annotation: "x"}}
	writeNoteFile(t, nbDir, ego)
	writeNoteFile(t, nbDir, up)

	esc := "\x1b["

	never, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--zones", "--color", "never", "--format", "text")
	if err != nil {
		t.Fatalf("color never: %v", err)
	}
	if strings.Contains(never, esc) {
		t.Errorf("property [4a]: --color never must not emit ANSI escapes")
	}

	always, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--zones", "--color", "always", "--format", "text")
	if err != nil {
		t.Fatalf("color always: %v", err)
	}
	if !strings.Contains(always, esc) {
		t.Errorf("property [4a]: --color always must emit ANSI escapes:\n%q", always)
	}

	js, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--zones", "--color", "always", "--format", "json")
	if err != nil {
		t.Fatalf("color always json: %v", err)
	}
	if strings.Contains(js, esc) {
		t.Errorf("property [4b]: json output must never contain ANSI escapes even with --color always")
	}
}
