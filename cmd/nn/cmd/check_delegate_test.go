package cmd

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Property: nn check delegates violation computation to the rules engine via
// engineCheckViolations, which returns the same pass/fail verdict as the legacy
// checkRepresentationGraph for a subgraph rooted at a given note, scoped to that
// subgraph (violations outside the root's reachable subgraph are excluded).

func TestEngineCheckViolations_MatchesLegacyVerdict(t *testing.T) {
	cases := []struct {
		name     string
		notes    []*note.Note
		rootID   string
		rep      string
		wantFail bool
	}{
		{
			name: "valid ontology",
			notes: []*note.Note{
				repNote("r", note.TypeModel, "ontology", note.Link{TargetID: "c", Type: "refines"}),
				repNote("c", note.TypeConcept, "ontology"),
			},
			rootID: "r", rep: "ontology", wantFail: false,
		},
		{
			name: "non-model root",
			notes: []*note.Note{
				repNote("r", note.TypeConcept, "ontology", note.Link{TargetID: "c", Type: "refines"}),
				repNote("c", note.TypeConcept, "ontology"),
			},
			rootID: "r", rep: "ontology", wantFail: true,
		},
		{
			name: "axiom missing grounded-by",
			notes: []*note.Note{
				repNote("r", note.TypeModel, "axiom", note.Link{TargetID: "c", Type: "refines"}),
				repNote("c", note.TypeConcept, "axiom"),
			},
			rootID: "r", rep: "axiom", wantFail: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			byID := map[string]*note.Note{}
			for _, n := range tc.notes {
				byID[n.ID] = n
			}
			legacy := checkRepresentationGraph(byID[tc.rootID], byID, tc.rep)
			engine := engineCheckViolations(byID[tc.rootID], tc.notes, tc.rep)

			if (len(legacy) > 0) != (len(engine) > 0) {
				t.Fatalf("verdict mismatch: legacy failed=%v engine failed=%v (legacy=%v engine=%v)",
					len(legacy) > 0, len(engine) > 0, legacy, engine)
			}
			if (len(engine) > 0) != tc.wantFail {
				t.Fatalf("engine verdict failed=%v, want %v (%v)", len(engine) > 0, tc.wantFail, engine)
			}
		})
	}
}

// Scoping property: a violation in a DIFFERENT representation subgraph must not
// be reported when checking the root's subgraph.
func TestEngineCheckViolations_ScopedToSubgraph(t *testing.T) {
	notes := []*note.Note{
		repNote("r", note.TypeModel, "ontology", note.Link{TargetID: "c", Type: "refines"}),
		repNote("c", note.TypeConcept, "ontology"),
		// A separate, broken taxonomy elsewhere in the notebook.
		repNote("bad", note.TypeConcept, "taxonomy"),
	}
	byID := map[string]*note.Note{}
	for _, n := range notes {
		byID[n.ID] = n
	}
	v := engineCheckViolations(byID["r"], notes, "ontology")
	if len(v) != 0 {
		t.Fatalf("root subgraph is clean; unrelated taxonomy must not leak: %v", v)
	}
}
