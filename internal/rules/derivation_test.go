package rules

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Property: built-in DERIVATION rules ship out of the box — a user need not
// write a fence to get transitively_governs / reachable. These are computed
// purely from the auto-exposed facts + builtin.dl.

func TestBuiltin_DerivesTransitivelyGoverns(t *testing.T) {
	notes := []*note.Note{
		{ID: "p", Type: note.TypeProtocol, Status: note.StatusPermanent,
			Links: []note.Link{{TargetID: "a", Type: "governs"}}},
		{ID: "a", Type: note.TypeConcept, Status: note.StatusReviewed,
			Links: []note.Link{{TargetID: "b", Type: "refines"}}},
		{ID: "b", Type: note.TypeConcept, Status: note.StatusReviewed},
	}

	e := NewEngine()
	for _, f := range FactsFromNotes(notes) {
		e.AddFact(f)
	}
	rs, err := ParseProgram(BuiltinRules())
	if err != nil {
		t.Fatalf("parse builtin: %v", err)
	}
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}

	got := queryKeys(e, "transitively_governs")
	// p governs a; a refines b ⇒ p transitively governs a and b.
	want := []string{"transitively_governs(p,a)", "transitively_governs(p,b)"}
	if !equalSlices(got, want) {
		t.Fatalf("transitively_governs = %v, want %v", got, want)
	}
}

func TestBuiltin_DerivesReachable(t *testing.T) {
	notes := []*note.Note{
		{ID: "x", Type: note.TypeConcept, Status: note.StatusReviewed,
			Links: []note.Link{{TargetID: "y", Type: "supports"}}},
		{ID: "y", Type: note.TypeConcept, Status: note.StatusReviewed,
			Links: []note.Link{{TargetID: "z", Type: "refines"}}},
		{ID: "z", Type: note.TypeConcept, Status: note.StatusReviewed},
	}

	e := NewEngine()
	for _, f := range FactsFromNotes(notes) {
		e.AddFact(f)
	}
	rs, _ := ParseProgram(BuiltinRules())
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}

	got := queryKeys(e, "reachable")
	want := []string{"reachable(x,y)", "reachable(x,z)", "reachable(y,z)"}
	if !equalSlices(got, want) {
		t.Fatalf("reachable = %v, want %v", got, want)
	}
}
