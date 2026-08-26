package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// ADR-0025: nn link rejects unknown relationship types without writing.
func TestLinkUnknownTypeRejected(t *testing.T) {
	nbDir, cfgFile := setupNotebookWithCfg(t)
	src := newTestNoteForCLI(note.GenerateID(), "From", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "To", note.TypeConcept)
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	_, _, err := executeWithStderr(t, cfgFile, "link", src.ID, dst.ID, "--annotation", "test", "--type", "bogus-type")
	if err == nil {
		t.Fatal("nn link unknown type: want error, got nil")
	}
	data, readErr := os.ReadFile(filepath.Join(nbDir, src.Filename()))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(data), dst.ID) {
		t.Errorf("unknown link type was written:\n%s", data)
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
