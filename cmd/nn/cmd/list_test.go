package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestListAll(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept))
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Beta", note.TypeArgument))

	out, err := execute("list")
	if err != nil {
		t.Fatalf("nn list: %v", err)
	}
	if !strings.Contains(out, "Alpha") || !strings.Contains(out, "Beta") {
		t.Errorf("list output missing notes: %q", out)
	}
}

func TestListFilterByType(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Concept Note", note.TypeConcept))
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Argument Note", note.TypeArgument))

	out, err := execute("list", "--type", "concept")
	if err != nil {
		t.Fatalf("nn list --type concept: %v", err)
	}
	if !strings.Contains(out, "Concept Note") {
		t.Errorf("output missing 'Concept Note': %q", out)
	}
	if strings.Contains(out, "Argument Note") {
		t.Errorf("output should not contain 'Argument Note': %q", out)
	}
}

func TestListFilterByStatus(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	draft := newTestNoteForCLI(note.GenerateID(), "Draft Note", note.TypeConcept)
	reviewed := newTestNoteForCLI(note.GenerateID(), "Reviewed Note", note.TypeConcept)
	reviewed.Status = note.StatusReviewed
	writeNoteFile(t, nbDir, draft)
	writeNoteFile(t, nbDir, reviewed)

	out, err := execute("list", "--status", "reviewed")
	if err != nil {
		t.Fatalf("nn list --status reviewed: %v", err)
	}
	if !strings.Contains(out, "Reviewed Note") {
		t.Errorf("output missing 'Reviewed Note': %q", out)
	}
	if strings.Contains(out, "Draft Note") {
		t.Errorf("output should not contain 'Draft Note': %q", out)
	}
}

func TestListFilterByTag(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	tagged := newTestNoteForCLI(note.GenerateID(), "Tagged", note.TypeConcept)
	tagged.Tags = []string{"zettelkasten"}
	untagged := newTestNoteForCLI(note.GenerateID(), "Untagged", note.TypeConcept)
	writeNoteFile(t, nbDir, tagged)
	writeNoteFile(t, nbDir, untagged)

	out, err := execute("list", "--tag", "zettelkasten")
	if err != nil {
		t.Fatalf("nn list --tag: %v", err)
	}
	if !strings.Contains(out, "Tagged") {
		t.Errorf("output missing 'Tagged': %q", out)
	}
	if strings.Contains(out, "Untagged") {
		t.Errorf("output should not contain 'Untagged': %q", out)
	}
}

func TestListOrphan(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	orphan := newTestNoteForCLI(note.GenerateID(), "Orphan", note.TypeConcept)
	linked := newTestNoteForCLI(note.GenerateID(), "Linked", note.TypeConcept)
	target := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeConcept)
	linked.Links = []note.Link{{TargetID: target.ID, Annotation: "relates to"}}
	writeNoteFile(t, nbDir, orphan)
	writeNoteFile(t, nbDir, linked)
	writeNoteFile(t, nbDir, target)

	out, err := execute("list", "--orphan")
	if err != nil {
		t.Fatalf("nn list --orphan: %v", err)
	}
	if !strings.Contains(out, "Orphan") {
		t.Errorf("output missing 'Orphan': %q", out)
	}
	if strings.Contains(out, "Linked") || strings.Contains(out, "Target") {
		t.Errorf("output should not contain linked notes: %q", out)
	}
}

func TestListJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "JSON Note", note.TypeConcept))

	out, err := execute("list", "--json")
	if err != nil {
		t.Fatalf("nn list --json: %v", err)
	}
	var result []map[string]any
	mustJSON(t, out, &result)
	if len(result) != 1 {
		t.Errorf("JSON list count = %d, want 1", len(result))
	}
}

func TestListLimit(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	for i := 0; i < 5; i++ {
		writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Note", note.TypeConcept))
	}
	out, err := execute("list", "--limit", "2", "--json")
	if err != nil {
		t.Fatalf("nn list --limit: %v", err)
	}
	var result []map[string]any
	mustJSON(t, out, &result)
	if len(result) != 2 {
		t.Errorf("limited list count = %d, want 2", len(result))
	}
}

// Assertion: --global returns only protocols with no outgoing governs links.
func TestListGlobalProtocols(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	global := newTestNoteForCLI(note.GenerateID(), "Global Protocol", note.TypeProtocol)
	contextual := newTestNoteForCLI(note.GenerateID(), "Contextual Protocol", note.TypeProtocol)
	target := newTestNoteForCLI(note.GenerateID(), "Target Note", note.TypeConcept)
	contextual.Links = []note.Link{{TargetID: target.ID, Annotation: "governs", Type: "governs"}}
	writeNoteFile(t, nbDir, global)
	writeNoteFile(t, nbDir, contextual)
	writeNoteFile(t, nbDir, target)

	out, err := execute("list", "--global")
	if err != nil {
		t.Fatalf("nn list --global: %v", err)
	}
	if !strings.Contains(out, "Global Protocol") {
		t.Errorf("expected global protocol in output:\n%s", out)
	}
	if strings.Contains(out, "Contextual Protocol") {
		t.Errorf("expected contextual protocol excluded from output:\n%s", out)
	}
}

// Assertion: --global excludes non-protocol notes.
func TestListGlobalExcludesNonProtocol(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	proto := newTestNoteForCLI(note.GenerateID(), "A Protocol", note.TypeProtocol)
	concept := newTestNoteForCLI(note.GenerateID(), "A Concept", note.TypeConcept)
	writeNoteFile(t, nbDir, proto)
	writeNoteFile(t, nbDir, concept)

	out, err := execute("list", "--global")
	if err != nil {
		t.Fatalf("nn list --global: %v", err)
	}
	if strings.Contains(out, "A Concept") {
		t.Errorf("expected non-protocol note excluded from --global output:\n%s", out)
	}
}

// Assertion: --global with --type non-protocol returns an error.
func TestListGlobalWithNonProtocolTypeErrors(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("list", "--global", "--type", "concept")
	if err == nil {
		t.Fatal("nn list --global --type concept: want error, got nil")
	}
}

// Assertion 1: --verbose flag is recognized (does not produce unknown flag error)
func TestListVerboseFlagExists(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Test Note", note.TypeConcept))

	// Should not error on unrecognized flag
	_, err := execute("list", "--verbose")
	if err != nil {
		t.Fatalf("nn list --verbose: %v", err)
	}
}

// Assertion 2: --verbose without --json produces expanded human-readable output including type, status, tags
func TestListVerboseTextOutput(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	note1 := newTestNoteForCLI(note.GenerateID(), "Concept Note", note.TypeConcept)
	note1.Status = note.StatusDraft
	note1.Tags = []string{"important", "review"}
	writeNoteFile(t, nbDir, note1)

	out, err := execute("list", "--verbose")
	if err != nil {
		t.Fatalf("nn list --verbose: %v", err)
	}

	// Verbose output should include type
	if !strings.Contains(out, "concept") {
		t.Errorf("verbose output missing type: %q", out)
	}
	// Verbose output should include status
	if !strings.Contains(out, "draft") {
		t.Errorf("verbose output missing status: %q", out)
	}
	// Verbose output should include tags
	if !strings.Contains(out, "important") || !strings.Contains(out, "review") {
		t.Errorf("verbose output missing tags: %q", out)
	}
}

// Assertion 3: --verbose with --json includes additional fields (type, status, tags, and created/modified timestamps)
func TestListVerboseJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	note1 := newTestNoteForCLI(note.GenerateID(), "Test Note", note.TypeArgument)
	note1.Status = note.StatusReviewed
	note1.Tags = []string{"validated"}
	writeNoteFile(t, nbDir, note1)

	out, err := execute("list", "--verbose", "--json")
	if err != nil {
		t.Fatalf("nn list --verbose --json: %v", err)
	}

	var result []map[string]any
	mustJSON(t, out, &result)
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	// Check that verbose JSON includes expected fields
	item := result[0]
	if item["type"] != "argument" {
		t.Errorf("expected type=argument, got %v", item["type"])
	}
	if item["status"] != "reviewed" {
		t.Errorf("expected status=reviewed, got %v", item["status"])
	}
	if tags, ok := item["tags"].([]any); !ok || len(tags) != 1 {
		t.Errorf("expected tags array with 1 element, got %v", item["tags"])
	}
	// Verbose JSON should include created and modified timestamps
	if _, ok := item["created"]; !ok {
		t.Error("verbose JSON missing 'created' field")
	}
	if _, ok := item["modified"]; !ok {
		t.Error("verbose JSON missing 'modified' field")
	}
}
