package rules

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// blockedChainNotes builds A --requires--> B --requires--> C, each with an open
// item unless overridden, so the built-in task-dependency rules can be exercised.
func blockedChainNotes(cDone bool) []*note.Note {
	cBody := "- [ ] do C"
	if cDone {
		cBody = "- [x] did C"
	}
	return []*note.Note{
		{ID: "a", Type: note.TypeConcept, Status: note.StatusReviewed,
			Body:  "- [ ] do A",
			Links: []note.Link{{TargetID: "b", Type: "requires"}}},
		{ID: "b", Type: note.TypeConcept, Status: note.StatusReviewed,
			Body:  "- [ ] do B",
			Links: []note.Link{{TargetID: "c", Type: "requires"}}},
		{ID: "c", Type: note.TypeConcept, Status: note.StatusReviewed, Body: cBody},
	}
}

func evalBuiltin(t *testing.T, notes []*note.Note) *Engine {
	t.Helper()
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
	return e
}

// property [2]: done(X) holds iff X has no open checkbox items.
func TestBuiltin_DerivesDone(t *testing.T) {
	e := evalBuiltin(t, blockedChainNotes(true)) // C done, A/B open
	got := queryKeys(e, "done")
	want := []string{"done(c)"}
	if !equalSlices(got, want) {
		t.Fatalf("done = %v, want %v", got, want)
	}
}

// property [1]: blocked(X) holds iff X requires a not-done target. When C is done,
// B is unblocked (its only dependency is satisfied) but A is blocked by B.
func TestBuiltin_DerivesBlocked_DirectOnly(t *testing.T) {
	e := evalBuiltin(t, blockedChainNotes(true)) // C done
	got := queryKeys(e, "blocked")
	want := []string{"blocked(a)"}
	if !equalSlices(got, want) {
		t.Fatalf("blocked = %v, want %v", got, want)
	}
}

// property [1] (transitive): when C is not done, B is blocked (requires C) and A
// is blocked transitively (requires B, which is blocked).
func TestBuiltin_DerivesBlocked_Transitive(t *testing.T) {
	e := evalBuiltin(t, blockedChainNotes(false)) // C not done
	got := queryKeys(e, "blocked")
	want := []string{"blocked(a)", "blocked(b)"}
	if !equalSlices(got, want) {
		t.Fatalf("blocked (transitive) = %v, want %v", got, want)
	}
}
