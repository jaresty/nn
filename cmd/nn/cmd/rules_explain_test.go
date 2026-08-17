package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Property: `nn rules explain <fact>` prints the derivation path of a derived
// fact down to base facts.
func TestRulesExplainShowsDerivation(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// p governs b (general); a refines b (a specializes b) ⇒ governance flows
	// general→specific, so transitively_governs(p,a) is derived by the built-in
	// derivation rules.
	p := newTestNoteForCLI(note.GenerateID(), "Protocol", note.TypeProtocol)
	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "B", note.TypeConcept)
	p.Links = []note.Link{{TargetID: b.ID, Annotation: "governs it", Type: "governs"}}
	a.Links = []note.Link{{TargetID: b.ID, Annotation: "refines it", Type: "refines"}}
	writeNoteFile(t, nbDir, p)
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	fact := "transitively_governs(" + p.ID + "," + a.ID + ")"
	out, err := execute("rules", "explain", fact)
	if err != nil {
		t.Fatalf("rules explain: %v\nout: %s", err, out)
	}
	// Explanation must name the target fact and mention "via rule" and a base fact.
	if !strings.Contains(out, fact) {
		t.Errorf("explanation missing target fact %q; got:\n%s", fact, out)
	}
	if !strings.Contains(out, "via rule") {
		t.Errorf("explanation missing rule provenance; got:\n%s", out)
	}
	if !strings.Contains(out, "base fact") {
		t.Errorf("explanation missing base fact; got:\n%s", out)
	}
}

func TestRulesExplainUnknownFactErrors(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("rules", "explain", "nope(x)")
	if err == nil {
		t.Fatal("expected error for a fact that was never derived")
	}
}
