package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: TestReviewRequiredActions — review output ends with a Required Actions section.
func TestReviewRequiredActions(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Orphan", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	out, err := execute("review")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if !strings.Contains(out, "## Required Actions") {
		t.Errorf("expected '## Required Actions' section in review output; got:\n%s", out)
	}
}

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

// Assertion: TestReviewLongNotesMarkdown — note with body > atomicityThreshold appears in Long notes section.
func TestReviewLongNotesMarkdown(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	longBody := strings.Repeat("x", atomicityThreshold+1)
	n := newTestNoteForCLI(note.GenerateID(), "HugeNote", note.TypeConcept)
	n.Body = longBody
	writeNoteFile(t, nbDir, n)

	out, err := execute("review")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if !strings.Contains(out, "Long notes:") {
		t.Errorf("expected 'Long notes:' section; got:\n%s", out)
	}
	if !strings.Contains(out, "HugeNote") {
		t.Errorf("expected 'HugeNote' in long notes list; got:\n%s", out)
	}
}

// Assertion: TestReviewLongNotesJSON — note with body > atomicityThreshold appears in JSON long_notes field.
func TestReviewLongNotesJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	longBody := strings.Repeat("x", atomicityThreshold+1)
	n := newTestNoteForCLI(note.GenerateID(), "HugeNote", note.TypeConcept)
	n.Body = longBody
	writeNoteFile(t, nbDir, n)

	out, err := execute("review", "--format", "json")
	if err != nil {
		t.Fatalf("review --format json: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	longNotes, ok := result["long_notes"]
	if !ok {
		t.Fatalf("expected 'long_notes' key in JSON; got keys from: %s", out)
	}
	items, ok := longNotes.([]any)
	if !ok || len(items) == 0 {
		t.Errorf("expected non-empty long_notes; got: %v", longNotes)
	}
}

// Assertion: TestReviewAgingNotesSection — review output contains aging and stale buckets
func TestReviewAgingNotesSection(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	aging := newTestNoteForCLI(note.GenerateID(), "Aging Note", note.TypeConcept)
	aging.Modified = time.Now().UTC().Add(-7 * 24 * time.Hour)
	writeNoteFile(t, nbDir, aging)

	stale := newTestNoteForCLI(note.GenerateID(), "Stale Note", note.TypeConcept)
	stale.Modified = time.Now().UTC().Add(-30 * 24 * time.Hour)
	writeNoteFile(t, nbDir, stale)

	out, err := execute("review")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if !strings.Contains(out, "aging (3–14 days)") {
		t.Errorf("want 'aging (3–14 days)' bucket in review output, got:\n%s", out)
	}
	if !strings.Contains(out, "stale (>14 days)") {
		t.Errorf("want 'stale (>14 days)' bucket in review output, got:\n%s", out)
	}
	if !strings.Contains(out, aging.ID) {
		t.Errorf("want aging note %s in review output", aging.ID)
	}
	if !strings.Contains(out, stale.ID) {
		t.Errorf("want stale note %s in review output", stale.ID)
	}
}

// Assertion: TestRequiredActionsHasFixCommands — each Required Actions item includes nn commands to fix it.
func TestRequiredActionsHasFixCommands(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// Create an orphan note (no links).
	orphan := newTestNoteForCLI(note.GenerateID(), "OrphanNote", note.TypeConcept)
	writeNoteFile(t, nbDir, orphan)

	// Create a dead-end: has outbound link but no inbound.
	target := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeConcept)
	deadEnd := newTestNoteForCLI(note.GenerateID(), "DeadEnd", note.TypeConcept)
	deadEnd.Links = []note.Link{{TargetID: target.ID, Type: "extends", Annotation: "extends target"}}
	writeNoteFile(t, nbDir, target)
	writeNoteFile(t, nbDir, deadEnd)

	// Create a long note.
	longNote := newTestNoteForCLI(note.GenerateID(), "LongNote", note.TypeConcept)
	longNote.Body = strings.Repeat("x", atomicityThreshold+1)
	writeNoteFile(t, nbDir, longNote)

	// Create an expired note.
	expiredTime := time.Now().UTC().Add(-24 * time.Hour)
	expiredNote := newTestNoteForCLI(note.GenerateID(), "ExpiredNote", note.TypeConcept)
	expiredNote.Expires = &expiredTime
	writeNoteFile(t, nbDir, expiredNote)

	// Create a friction candidate.
	frictionNote := newTestNoteForCLI(note.GenerateID(), "FrictionNote", note.TypeObservation)
	frictionNote.Tags = []string{"friction-candidate"}
	writeNoteFile(t, nbDir, frictionNote)

	out, err := execute("review")
	if err != nil {
		t.Fatalf("review: %v", err)
	}

	cases := []struct {
		label   string
		wantCmd string
	}{
		{"orphan fix", "nn suggest-links"},
		{"dead-end fix", "nn link"},
		{"long note fix", "nn show"},
		{"expired note fix", "nn delete"},
		{"friction candidate fix", "nn new --type protocol"},
	}
	for _, c := range cases {
		if !strings.Contains(out, c.wantCmd) {
			t.Errorf("[%s] expected %q in Required Actions output; got:\n%s", c.label, c.wantCmd, out)
		}
	}
}
