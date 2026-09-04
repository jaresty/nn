package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Property N1: nn new --aliases "A,B" sets Note.Aliases to the comma-split,
// trimmed, empty-dropped list.
func TestNewAliases(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("new", "--title", "Test-driven development", "--type", "concept",
		"--aliases", "TDD, TDDev ,", "--content", "body", "--no-edit")
	if err != nil {
		t.Fatalf("nn new --aliases: %v", err)
	}
	id := strings.TrimPrefix(strings.TrimSpace(out), "created ")
	shown, _ := execute("show", id)
	if !strings.Contains(shown, "TDD") || !strings.Contains(shown, "TDDev") {
		t.Errorf("property N1: aliases not set on new note: %q", shown)
	}
}

// Property U1: nn update --aliases "A,B" replaces the alias set.
func TestUpdateAliasesReplaces(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Test-driven development", note.TypeConcept)
	n.Aliases = []string{"OLD"}
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--aliases", "TDD", "--since", sinceFor(n), "--no-edit")
	if err != nil {
		t.Fatalf("nn update --aliases: %v", err)
	}
	shown, _ := execute("show", n.ID)
	if !strings.Contains(shown, "TDD") {
		t.Errorf("property U1: new alias TDD absent: %q", shown)
	}
	if strings.Contains(shown, "OLD") {
		t.Errorf("property U1: --aliases did not replace; OLD still present: %q", shown)
	}
}

// Property U2a + U2b + U2c: --aliases-add adds, --aliases-remove removes,
// unmentioned prior aliases are preserved.
func TestUpdateAliasesAddRemove(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Test-driven development", note.TypeConcept)
	n.Aliases = []string{"KEEP", "DROP"}
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--aliases-add", "ADD", "--aliases-remove", "DROP",
		"--since", sinceFor(n), "--no-edit")
	if err != nil {
		t.Fatalf("nn update --aliases-add/-remove: %v", err)
	}
	shown, _ := execute("show", n.ID)
	if !strings.Contains(shown, "ADD") {
		t.Errorf("property U2a: added alias ADD absent: %q", shown)
	}
	if strings.Contains(shown, "DROP") {
		t.Errorf("property U2b: removed alias DROP still present: %q", shown)
	}
	if !strings.Contains(shown, "KEEP") {
		t.Errorf("property U2c: preserved alias KEEP absent: %q", shown)
	}
}

// Property E1: an update with only alias flags is not rejected by the
// required-flag guard.
func TestUpdateAliasesOnlyNotRejected(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Test-driven development", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--aliases-add", "TDD", "--since", sinceFor(n), "--no-edit")
	if err != nil {
		t.Fatalf("property E1: alias-only update rejected: %v", err)
	}
}
