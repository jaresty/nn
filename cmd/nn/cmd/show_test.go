package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestShowNote(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Show Me", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if !strings.Contains(out, "Show Me") {
		t.Errorf("output %q does not contain title 'Show Me'", out)
	}
}

func TestShowNoteNotFound(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("show", "99999999999999-0000")
	if err == nil {
		t.Fatal("nn show nonexistent: want error, got nil")
	}
}

// Assertion: TestShowProtocolAppendsDerivationInstruction — plain nn show on a protocol note includes ## Protocols block.
func TestShowProtocolAppendsDerivationInstruction(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Protocol", note.TypeProtocol)
	n.Body = "Do the thing before acting."
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if !strings.Contains(out, "## Protocols") {
		t.Errorf("expected '## Protocols' derivation block in protocol note output; got:\n%s", out)
	}
}

// Assertion: TestShowNonProtocolNoDerivation — nn show on a concept note does NOT include ## Protocols block.
func TestShowNonProtocolNoDerivation(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	proto := newTestNoteForCLI(note.GenerateID(), "My Protocol", note.TypeProtocol)
	proto.Body = "Do the thing."
	concept := newTestNoteForCLI(note.GenerateID(), "My Concept", note.TypeConcept)
	concept.Body = "A concept about things."
	writeNoteFile(t, nbDir, proto)
	writeNoteFile(t, nbDir, concept)

	out, err := execute("show", concept.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if strings.Contains(out, "## Protocols") {
		t.Errorf("expected no '## Protocols' block for non-protocol note; got:\n%s", out)
	}
}

// Assertion: TestShowProtocolJSONNoDerivation — --json output does NOT include the derivation text.
func TestShowProtocolJSONNoDerivation(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Protocol", note.TypeProtocol)
	n.Body = "Do the thing before acting."
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID, "--json")
	if err != nil {
		t.Fatalf("nn show --json: %v", err)
	}
	if strings.Contains(out, "## Protocols") {
		t.Errorf("expected no derivation block in JSON output; got:\n%s", out)
	}
}
