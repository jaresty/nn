package rules

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [3]: governs_note(P, X) holds for direct governs, whole-tree
// representation governance (P governs the rep root ⇒ governs every node in the
// subtree), and refines/extends specialization chains.
func TestBuiltin_GovernsNote_AllModels(t *testing.T) {
	notes := []*note.Note{
		// Direct: P governs D.
		{ID: "p", Type: note.TypeProtocol, Status: note.StatusPermanent,
			Links: []note.Link{
				{TargetID: "d", Type: "governs"},   // direct
				{TargetID: "root", Type: "governs"}, // whole-tree root
				{TargetID: "spec0", Type: "governs"}, // specialization base
			}},
		{ID: "d", Type: note.TypeConcept, Status: note.StatusReviewed},

		// Whole-tree: root + child share a representation, joined by a non-refines
		// edge (supports) so only the whole-tree rule (not specialization) catches it.
		{ID: "root", Type: note.TypeModel, Status: note.StatusReviewed, Representation: "ontology",
			Links: []note.Link{{TargetID: "child", Type: "supports"}}},
		{ID: "child", Type: note.TypeConcept, Status: note.StatusReviewed, Representation: "ontology"},

		// Specialization: spec1 refines spec0 (no representation involved).
		{ID: "spec0", Type: note.TypeConcept, Status: note.StatusReviewed},
		{ID: "spec1", Type: note.TypeConcept, Status: note.StatusReviewed,
			Links: []note.Link{{TargetID: "spec0", Type: "refines"}}},
	}

	e := evalBuiltin(t, notes)
	got := queryKeys(e, "governs_note")

	// Each of these must be derived.
	wantContains := []string{
		"governs_note(p,d)",     // direct
		"governs_note(p,root)",  // direct (root is a direct governs target too)
		"governs_note(p,child)", // whole-tree: child in root's ontology subtree
		"governs_note(p,spec0)", // direct
		"governs_note(p,spec1)", // specialization: spec1 refines spec0 which p governs
	}
	set := map[string]bool{}
	for _, k := range got {
		set[k] = true
	}
	for _, w := range wantContains {
		if !set[w] {
			t.Errorf("governs_note missing %q; got %v", w, got)
		}
	}
}
