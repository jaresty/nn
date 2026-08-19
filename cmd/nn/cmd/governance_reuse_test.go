package cmd

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// governanceTestNotes builds a small notebook: protocol P governs note A
// directly, and A is refined by B, so transitively_governs propagates P to B.
func governanceTestNotes() []*note.Note {
	return []*note.Note{
		{ID: "p1", Title: "Protocol One", Type: note.TypeProtocol, Status: "permanent",
			Links: []note.Link{{TargetID: "a1", Type: "governs", Annotation: "g"}}},
		{ID: "a1", Title: "Governed A", Type: note.TypeConcept, Status: "reviewed"},
		{ID: "b1", Title: "Refiner B", Type: note.TypeConcept, Status: "reviewed",
			Links: []note.Link{{TargetID: "a1", Type: "refines", Annotation: "r"}}},
		{ID: "c1", Title: "Unrelated C", Type: note.TypeConcept, Status: "draft"},
	}
}

// property [22]: governedBy(buildGovernanceEngine(all), id) returns the same
// protocols (same IDs, same order) as evaluating per note. Building the engine
// once and querying per note must not change governance results.
func TestGovernedByMatchesPerNoteEval(t *testing.T) {
	all := governanceTestNotes()
	engine := buildGovernanceEngine(all)

	cases := map[string][]string{
		"a1": {"p1"}, // directly governed
		"b1": {"p1"}, // governed via refines chain
		"c1": nil,    // ungoverned
	}
	for id, want := range cases {
		got := governedBy(engine, id)
		var gotIDs []string
		for _, p := range got {
			gotIDs = append(gotIDs, p.ID)
		}
		if len(gotIDs) != len(want) {
			t.Fatalf("governedBy(%q) = %v, want %v", id, gotIDs, want)
		}
		for i := range want {
			if gotIDs[i] != want[i] {
				t.Fatalf("governedBy(%q)[%d] = %q, want %q", id, i, gotIDs[i], want[i])
			}
		}
	}
}

// property [23]: rendering M notes runs Eval exactly once. buildGovernanceEngine
// is the single place Eval happens; querying it per note must not re-Eval.
// evalCount on the engine records how many times Eval ran.
func TestBuildGovernanceEngineEvalsOnce(t *testing.T) {
	all := governanceTestNotes()
	engine := buildGovernanceEngine(all)
	// Query for several notes — none of these may trigger another Eval.
	_ = governedBy(engine, "a1")
	_ = governedBy(engine, "b1")
	_ = governedBy(engine, "c1")
	if n := engine.engine.EvalCount(); n != 1 {
		t.Fatalf("Eval ran %d times building+querying, want exactly 1", n)
	}
}
