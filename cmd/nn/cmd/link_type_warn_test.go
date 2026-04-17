package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: nn link --type <unknown> prints warning to stderr, exit 0.
func TestLinkUnknownTypeWarns(t *testing.T) {
	nbDir, cfgFile := setupNotebookWithCfg(t)
	src := newTestNoteForCLI(note.GenerateID(), "From", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "To", note.TypeConcept)
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	_, stderr, err := executeWithStderr(t, cfgFile, "link", src.ID, dst.ID, "--annotation", "test", "--type", "bogus-type")
	if err != nil {
		t.Fatalf("nn link unknown type should exit 0: %v", err)
	}
	if !strings.Contains(stderr, "warning") {
		t.Errorf("expected warning for unknown link type, got stderr: %q", stderr)
	}
}

// Assertion: nn link --type <known> produces no warning.
func TestLinkKnownTypeNoWarn(t *testing.T) {
	nbDir, cfgFile := setupNotebookWithCfg(t)
	src := newTestNoteForCLI(note.GenerateID(), "From", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "To", note.TypeConcept)
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	_, stderr, err := executeWithStderr(t, cfgFile, "link", src.ID, dst.ID, "--annotation", "test", "--type", "refines")
	if err != nil {
		t.Fatalf("nn link known type: %v", err)
	}
	if strings.Contains(stderr, "warning") {
		t.Errorf("unexpected warning for known type 'refines': %q", stderr)
	}
}
