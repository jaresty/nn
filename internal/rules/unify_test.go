package rules

import "testing"

// A body literal that repeats a variable within the SAME atom — p(X, X) —
// constrains both argument positions to be equal. This exercises unify's
// within-atom pending-binding path (distinct from a cross-atom join, where the
// repeated variable is already bound by an earlier literal). The predicate-index
// refactor routes this case through the `pending` slice, so it needs direct
// coverage.
func TestUnify_RepeatedVarWithinAtomMatchesEqualArgs(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "p", Args: []string{"a", "a"}}) // both positions equal
	e.AddFact(Fact{Pred: "p", Args: []string{"a", "b"}}) // positions differ

	// m(X) holds iff there is a p-fact whose two arguments are identical.
	rs := mustParse(t, `m(X) :- p(X, X).`)
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}

	got := queryKeys(e, "m")
	want := []string{"m(a)"} // only p(a,a) matches; p(a,b) must be rejected
	if !equalSlices(got, want) {
		t.Fatalf("m = %v, want %v (p(a,a) must match, p(a,b) must not)", got, want)
	}
}
