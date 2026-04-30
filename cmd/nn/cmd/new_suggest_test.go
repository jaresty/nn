package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestNewPrintsSuggestionsAfterWrite(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// Add similar notes so suggestions have something to find.
	sim1 := newTestNoteForCLI(note.GenerateID(), "Hook latency tradeoffs", note.TypeConcept)
	sim1.Body = "Stop hook agents add latency after every response turn in Claude Code."
	sim1.Tags = []string{"hooks"}
	sim2 := newTestNoteForCLI(note.GenerateID(), "PreCompact hook limitations", note.TypeObservation)
	sim2.Body = "PreCompact does not support type:agent hooks — stop hooks work instead."
	sim2.Tags = []string{"hooks"}
	writeNoteFile(t, nbDir, sim1)
	writeNoteFile(t, nbDir, sim2)
	commitNoteFile(t, nbDir, sim1)
	commitNoteFile(t, nbDir, sim2)

	out, err := execute("new",
		"--title", "Stop hook design",
		"--type", "concept",
		"--content", "The stop hook fires after each Claude response and runs a shell script.",
		"--no-edit",
	)
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	if !strings.Contains(out, "Suggestions") {
		t.Errorf("nn new: expected 'Suggestions' in output, got %q", out)
	}
}

func TestNewNoSuggestSuppressesOutput(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	sim1 := newTestNoteForCLI(note.GenerateID(), "Hook latency tradeoffs", note.TypeConcept)
	sim1.Body = "Stop hook agents add latency after every response turn in Claude Code."
	sim1.Tags = []string{"hooks"}
	writeNoteFile(t, nbDir, sim1)

	out, err := execute("new",
		"--title", "Stop hook design",
		"--type", "concept",
		"--content", "The stop hook fires after each Claude response.",
		"--no-edit",
		"--no-suggest",
	)
	if err != nil {
		t.Fatalf("nn new --no-suggest: %v", err)
	}
	if strings.Contains(out, "Suggestions") {
		t.Errorf("nn new --no-suggest: 'Suggestions' should be suppressed, got %q", out)
	}
}
