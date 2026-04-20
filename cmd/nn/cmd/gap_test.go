package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: TestGapCommandExists — command is registered and runs without error.
func TestGapCommandExists(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("gap", "zettelkasten")
	if err != nil {
		t.Fatalf("gap command failed: %v", err)
	}
}

// Assertion: TestGapSearchResultsSection — output contains notes matching the topic.
func TestGapSearchResultsSection(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	match := writeSuggestNote(t, nbDir, "20240101000001", "Zettelkasten Overview", "zettelkasten is a method for linking notes")
	writeSuggestNote(t, nbDir, "20240101000002", "Cooking Recipes", "pasta carbonara recipe")

	out, err := execute("gap", "zettelkasten")
	if err != nil {
		t.Fatalf("gap: %v", err)
	}
	if !strings.Contains(out, "## Topic notes") {
		t.Errorf("expected '## Topic notes' section; got:\n%s", out)
	}
	if !strings.Contains(out, match.ID) {
		t.Errorf("expected matching note %q in output; got:\n%s", match.ID, out)
	}
	if strings.Contains(out, "20240101000002") {
		t.Errorf("expected non-matching note excluded from topic notes; got:\n%s", out)
	}
}

// Assertion: TestGapNeighborhoodSection — output contains linked neighbors of topic notes.
func TestGapNeighborhoodSection(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	neighbor := writeSuggestNote(t, nbDir, "20240101000003", "Atomic Notes", "atomic notes are single-idea units")

	match := newTestNoteForCLI("20240101000001", "Zettelkasten Overview", note.TypeConcept)
	match.Body = "zettelkasten is a method for linking notes"
	match.Links = []note.Link{{TargetID: neighbor.ID, Type: "extends", Annotation: "extends atomic notes"}}
	writeNoteFile(t, nbDir, match)

	writeSuggestNote(t, nbDir, "20240101000002", "Cooking Recipes", "pasta carbonara recipe")

	out, err := execute("gap", "zettelkasten")
	if err != nil {
		t.Fatalf("gap: %v", err)
	}
	if !strings.Contains(out, "## Neighborhood") {
		t.Errorf("expected '## Neighborhood' section; got:\n%s", out)
	}
	if !strings.Contains(out, neighbor.ID) {
		t.Errorf("expected neighbor %q in neighborhood section; got:\n%s", neighbor.ID, out)
	}
}

// Assertion: TestGapFormatJSON — --format json produces valid JSON with required keys.
func TestGapFormatJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeSuggestNote(t, nbDir, "20240101000001", "Zettelkasten Overview", "zettelkasten is a method for linking notes")

	out, err := execute("gap", "zettelkasten", "--format", "json")
	if err != nil {
		t.Fatalf("gap --format json: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("expected valid JSON; got:\n%s\nerr: %v", out, err)
	}
	for _, key := range []string{"topic_notes", "neighbors"} {
		if _, ok := result[key]; !ok {
			t.Errorf("expected %q key in JSON; got keys: %v", key, jsonKeys(result))
		}
	}
}

// Assertion: TestGapLimit — --limit caps the number of topic notes returned.
func TestGapLimit(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeSuggestNote(t, nbDir, "20240101000001", "Zettelkasten Overview", "zettelkasten links notes")
	writeSuggestNote(t, nbDir, "20240101000002", "Zettelkasten Principles", "zettelkasten atomic permanence")
	writeSuggestNote(t, nbDir, "20240101000003", "Zettelkasten Practice", "zettelkasten daily writing practice")

	out, err := execute("gap", "zettelkasten", "--limit", "1")
	if err != nil {
		t.Fatalf("gap --limit 1: %v", err)
	}
	// Count note IDs in the Topic notes section (before ## Neighborhood).
	topicSection := out
	if idx := strings.Index(out, "## Neighborhood"); idx >= 0 {
		topicSection = out[:idx]
	}
	// Each topic note appears as "### <id>" in the section.
	count := strings.Count(topicSection, "\n### ")
	if count > 1 {
		t.Errorf("expected at most 1 topic note with --limit 1; got %d in:\n%s", count, topicSection)
	}
}
