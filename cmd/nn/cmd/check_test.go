package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [5]: nn check <id> reads representation from frontmatter and validates structural contract.
func TestCheckValidatesRepresentation(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "Domain Ontology", note.TypeModel)
	n.Representation = "ontology"
	n.Body = "## Concepts\n\n- thing\n\n## Relations\n\n- is-a"
	writeNoteFile(t, nbDir, n)

	out, err := execute("check", n.ID)
	if err != nil {
		t.Fatalf("nn check %s: %v\n%s", n.ID, err, out)
	}
	if !strings.Contains(out, "ok") && !strings.Contains(out, "pass") {
		t.Errorf("nn check output missing ok/pass:\n%s", out)
	}
}

// property [5b]: nn check <id> exits non-zero when note fails validation.
func TestCheckFailsInvalidRepresentation(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "Empty Ontology", note.TypeModel)
	n.Representation = "ontology"
	n.Body = "no required sections here"
	writeNoteFile(t, nbDir, n)

	_, err := execute("check", n.ID)
	if err == nil {
		t.Errorf("nn check expected non-zero exit for invalid ontology, got nil error")
	}
}

// property [6]: nn check <id> --as <value> overrides frontmatter representation.
func TestCheckAsOverride(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "Unlabeled Ontology", note.TypeModel)
	n.Body = "no required sections here"
	writeNoteFile(t, nbDir, n)

	_, err := execute("check", n.ID, "--as", "ontology")
	if err == nil {
		t.Errorf("nn check --as ontology expected non-zero exit for invalid body, got nil error")
	}
}

// property [7]: nn check <id> --set-representation stamps representation on passing note.
func TestCheckSetRepresentation(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "Valid Ontology", note.TypeModel)
	n.Body = "## Concepts\n\n- thing\n\n## Relations\n\n- is-a"
	writeNoteFile(t, nbDir, n)

	out, err := execute("check", n.ID, "--as", "ontology", "--set-representation")
	if err != nil {
		t.Fatalf("nn check --set-representation: %v\n%s", n.ID, out)
	}

	shown, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show %s: %v", n.ID, err)
	}
	if !strings.Contains(shown, "representation: ontology") {
		t.Errorf("nn show after --set-representation missing field:\n%s", shown)
	}
}
