package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: nn show --global emits applies_when for protocols that have it.
func TestShowGlobalEmitsAppliesWhen(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Global Protocol", note.TypeProtocol)
	n.AppliesWhen = "before any external lookup"
	n.Body = "Full body text that should not appear in global output."
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "applies_when") {
		t.Errorf("expected applies_when in --global output; got:\n%s", out)
	}
}

// Assertion: nn show --global does NOT emit full body of protocol notes.
func TestShowGlobalNoFullBody(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Global Protocol", note.TypeProtocol)
	n.AppliesWhen = "before any external lookup"
	n.Body = "UNIQUE_BODY_SENTINEL_TEXT_XYZ"
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if strings.Contains(out, "UNIQUE_BODY_SENTINEL_TEXT_XYZ") {
		t.Errorf("expected full body to be absent from --global output; got:\n%s", out)
	}
}
