package cmd

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
	"github.com/jaresty/nn/internal/rules"
)

// This is the ground+falsify guard the ADR requires: the new Datalog engine and
// the existing checkRepresentationGraph must AGREE on whether a representation
// subgraph is valid. It runs the real checkRepresentationGraph (authoritative
// current behavior) and the engine on the same fixtures and asserts that when
// check reports violations, the engine also reports at least one violation for a
// node in that subgraph — and when check is clean, the root's subgraph is clean.
//
// The FAIL this fires against: absence of a differential agreement guard between
// the two implementations.

func engineViolationIDs(t *testing.T, notes []*note.Note) map[string]bool {
	t.Helper()
	e := rules.NewEngine()
	for _, f := range rules.FactsFromNotes(notes) {
		e.AddFact(f)
	}
	rs, err := rules.ParseProgram(rules.BuiltinRules())
	if err != nil {
		t.Fatalf("parse builtin: %v", err)
	}
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}
	ids := map[string]bool{}
	for _, f := range e.Query("violation") {
		if len(f.Args) >= 1 {
			ids[f.Args[0]] = true
		}
	}
	return ids
}

func repNote(id string, ty note.Type, rep string, links ...note.Link) *note.Note {
	return &note.Note{ID: id, Type: ty, Status: note.StatusReviewed, Representation: rep, Links: links}
}

func TestRulesEngineAgreesWithCheck(t *testing.T) {
	cases := []struct {
		name     string
		notes    []*note.Note
		rootID   string
		rep      string
		wantFail bool // does checkRepresentationGraph report violations?
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
			name: "taxonomy bad link",
			notes: []*note.Note{
				repNote("r", note.TypeModel, "taxonomy", note.Link{TargetID: "c", Type: "supports"}),
				repNote("c", note.TypeConcept, "taxonomy"),
			},
			rootID: "r", rep: "taxonomy", wantFail: true,
		},
		{
			name: "axiom missing grounded-by",
			notes: []*note.Note{
				repNote("r", note.TypeModel, "axiom", note.Link{TargetID: "c", Type: "refines"}),
				repNote("c", note.TypeConcept, "axiom"),
			},
			rootID: "r", rep: "axiom", wantFail: true,
		},
		{
			name: "cycle",
			notes: []*note.Note{
				repNote("r", note.TypeModel, "ontology", note.Link{TargetID: "c", Type: "refines"}),
				repNote("c", note.TypeConcept, "ontology", note.Link{TargetID: "r", Type: "refines"}),
			},
			rootID: "r", rep: "ontology", wantFail: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			byID := map[string]*note.Note{}
			for _, n := range tc.notes {
				byID[n.ID] = n
			}
			// Authoritative current behavior.
			checkViols := checkRepresentationGraph(byID[tc.rootID], byID, tc.rep)
			checkFailed := len(checkViols) > 0
			if checkFailed != tc.wantFail {
				t.Fatalf("fixture assumption wrong: checkRepresentationGraph failed=%v, want %v (%v)", checkFailed, tc.wantFail, checkViols)
			}

			// Engine must agree on the pass/fail verdict for the root's subgraph.
			engineIDs := engineViolationIDs(t, tc.notes)
			engineFlaggedInSubgraph := false
			for id := range engineIDs {
				if _, ok := byID[id]; ok {
					engineFlaggedInSubgraph = true
					break
				}
			}
			if checkFailed && !engineFlaggedInSubgraph {
				t.Errorf("check failed but engine reported no violation for the subgraph; check=%v", checkViols)
			}
			if !checkFailed && engineFlaggedInSubgraph {
				t.Errorf("check passed but engine flagged a violation: %v", engineIDs)
			}
		})
	}
}
