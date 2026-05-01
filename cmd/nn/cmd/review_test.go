package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: TestReviewCommandExists — command is registered and runs without error.
func TestReviewCommandExists(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("review")
	if err != nil {
		t.Fatalf("review command failed: %v", err)
	}
}

// Assertion: TestReviewGrowthSection — output contains growth stats block.
func TestReviewGrowthSection(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n1 := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	n2 := newTestNoteForCLI(note.GenerateID(), "Beta", note.TypeQuestion)
	writeNoteFile(t, nbDir, n1)
	writeNoteFile(t, nbDir, n2)

	out, err := execute("review")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if !strings.Contains(out, "## Growth") {
		t.Errorf("expected '## Growth' section; got:\n%s", out)
	}
	if !strings.Contains(out, "Total notes:") {
		t.Errorf("expected 'Total notes:' in growth section; got:\n%s", out)
	}
}

// Assertion: TestReviewConnectivitySection — output contains connectivity stats.
func TestReviewConnectivitySection(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n1 := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	n2 := newTestNoteForCLI(note.GenerateID(), "Beta", note.TypeConcept)
	n1.Links = []note.Link{{TargetID: n2.ID, Type: "extends", Annotation: "alpha extends beta"}}
	writeNoteFile(t, nbDir, n1)
	writeNoteFile(t, nbDir, n2)

	out, err := execute("review")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if !strings.Contains(out, "## Connectivity") {
		t.Errorf("expected '## Connectivity' section; got:\n%s", out)
	}
	if !strings.Contains(out, "Orphans:") {
		t.Errorf("expected 'Orphans:' in connectivity section; got:\n%s", out)
	}
	if !strings.Contains(out, "Dead-ends:") {
		t.Errorf("expected 'Dead-ends:' in connectivity section; got:\n%s", out)
	}
}

// Assertion: TestReviewFormatJSON — --format json produces valid JSON with required keys.
func TestReviewFormatJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	out, err := execute("review", "--format", "json")
	if err != nil {
		t.Fatalf("review --format json: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("expected valid JSON; got:\n%s\nerr: %v", out, err)
	}
	for _, key := range []string{"growth", "connectivity"} {
		if _, ok := result[key]; !ok {
			t.Errorf("expected %q key in JSON; got keys: %v", key, jsonKeys(result))
		}
	}
}

// Assertion: TestReviewDeadEndDetection — note with only outbound links (no inbound) is a dead-end.
func TestReviewDeadEndDetection(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	target := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeConcept)
	// deadEnd has outgoing link but nothing links to it.
	deadEnd := newTestNoteForCLI(note.GenerateID(), "DeadEnd", note.TypeConcept)
	deadEnd.Links = []note.Link{{TargetID: target.ID, Type: "extends", Annotation: "dead end extends target"}}
	// isolated has no links at all — it's an orphan, not a dead-end.
	isolated := newTestNoteForCLI(note.GenerateID(), "Isolated", note.TypeConcept)
	writeNoteFile(t, nbDir, target)
	writeNoteFile(t, nbDir, deadEnd)
	writeNoteFile(t, nbDir, isolated)

	out, err := execute("review")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	// deadEnd should appear in dead-ends list
	if !strings.Contains(out, deadEnd.ID) {
		t.Errorf("expected dead-end note %q in review output; got:\n%s", deadEnd.ID, out)
	}
}

// Assertion: TestReviewGlobalProtocolNotOrphan — global notes (type=protocol, status=permanent) are excluded from orphan list.
func TestReviewGlobalProtocolNotOrphan(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	global := newTestNoteForCLI(note.GenerateID(), "My Protocol", note.TypeProtocol)
	global.Status = note.StatusPermanent
	// no links — would be an orphan if not filtered
	writeNoteFile(t, nbDir, global)

	out, err := execute("review")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if strings.Contains(out, global.ID) {
		t.Errorf("global protocol note %q must not appear in orphan list; got:\n%s", global.ID, out)
	}
}

// Assertion: TestReviewRecentNotes — notes created in last 7 days are counted.
func TestReviewRecentNotes(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	recent := newTestNoteForCLI(note.GenerateID(), "Recent", note.TypeConcept)
	recent.Created = time.Now().UTC()
	old := newTestNoteForCLI(note.GenerateID(), "Old", note.TypeConcept)
	old.Created = time.Now().UTC().AddDate(0, 0, -30)
	writeNoteFile(t, nbDir, recent)
	writeNoteFile(t, nbDir, old)

	out, err := execute("review")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if !strings.Contains(out, "last 7 days") {
		t.Errorf("expected 'last 7 days' in growth section; got:\n%s", out)
	}
}
