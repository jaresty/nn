package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestSuggestTagsReturnsTagsFromSimilarNotes(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// Target note: about hooks, no tags yet.
	target := newTestNoteForCLI(note.GenerateID(), "Stop hook design", note.TypeConcept)
	target.Body = "The stop hook fires after each Claude response and runs a shell script."
	target.Tags = []string{}

	// Similar notes with tags.
	sim1 := newTestNoteForCLI(note.GenerateID(), "Hook latency tradeoffs", note.TypeConcept)
	sim1.Body = "Stop hook agents add latency after every response turn."
	sim1.Tags = []string{"hooks", "architecture"}

	sim2 := newTestNoteForCLI(note.GenerateID(), "PreCompact hook limitations", note.TypeObservation)
	sim2.Body = "PreCompact does not support type:agent hooks — only command hooks fire."
	sim2.Tags = []string{"hooks", "claude-code"}

	// Dissimilar note with unrelated tag.
	other := newTestNoteForCLI(note.GenerateID(), "Zettelkasten note types", note.TypeConcept)
	other.Body = "Notes in a Zettelkasten are atomic and linked by specific relationships."
	other.Tags = []string{"zettelkasten"}

	writeNoteFile(t, nbDir, target)
	writeNoteFile(t, nbDir, sim1)
	writeNoteFile(t, nbDir, sim2)
	writeNoteFile(t, nbDir, other)

	out, err := execute("suggest-tags", target.ID)
	if err != nil {
		t.Fatalf("nn suggest-tags: %v", err)
	}
	// "hooks" appears in 2 similar notes — should be suggested.
	if !strings.Contains(out, "hooks") {
		t.Errorf("suggest-tags: expected 'hooks' in output, got %q", out)
	}
}

func TestSuggestTagsJSONOutput(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	target := newTestNoteForCLI(note.GenerateID(), "Stop hook design", note.TypeConcept)
	target.Body = "The stop hook fires after each Claude response and runs a shell script."
	target.Tags = []string{}

	sim1 := newTestNoteForCLI(note.GenerateID(), "Hook latency tradeoffs", note.TypeConcept)
	sim1.Body = "Stop hook agents add latency after every response turn."
	sim1.Tags = []string{"hooks", "architecture"}

	sim2 := newTestNoteForCLI(note.GenerateID(), "PreCompact hook limitations", note.TypeObservation)
	sim2.Body = "PreCompact does not support type:agent hooks — only command hooks fire."
	sim2.Tags = []string{"hooks", "claude-code"}

	writeNoteFile(t, nbDir, target)
	writeNoteFile(t, nbDir, sim1)
	writeNoteFile(t, nbDir, sim2)

	out, err := execute("suggest-tags", target.ID, "--json")
	if err != nil {
		t.Fatalf("nn suggest-tags --json: %v", err)
	}
	var result []struct {
		Tag       string   `json:"tag"`
		FromNotes []string `json:"from_notes"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("nn suggest-tags --json: invalid JSON: %v\n%s", err, out)
	}
	found := false
	for _, r := range result {
		if r.Tag == "hooks" && len(r.FromNotes) >= 2 {
			found = true
		}
	}
	if !found {
		t.Errorf("suggest-tags --json: expected hooks with ≥2 from_notes, got %v", result)
	}
}
