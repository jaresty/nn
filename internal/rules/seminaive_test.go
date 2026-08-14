package rules

import (
	"fmt"
	"slices"
	"testing"
)

// A deep recursive chain exercises the semi-naive fixpoint across many rounds:
// each round derives reachable facts one hop further, so a delta transform that
// fails to feed newly-derived facts back into the next round would stop early
// and drop the far transitive pairs. The exact expected closure size pins that.
func TestSemiNaive_DeepChainClosureComplete(t *testing.T) {
	const n = 8 // chain a0 -> a1 -> ... -> a7
	e := NewEngine()
	for i := 0; i+1 < n; i++ {
		e.AddFact(Fact{Pred: "link", Args: []string{fmt.Sprintf("a%d", i), fmt.Sprintf("a%d", i+1), "refines"}})
	}
	rs := mustParse(t,
		`reach(X, Y) :- link(X, Y, _).`,
		`reach(X, Z) :- reach(X, Y), link(Y, Z, _).`,
	)
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}

	// Transitive closure of a linear chain of n nodes: every pair (i, j) with
	// i < j, i.e. n*(n-1)/2 facts.
	got := len(queryKeys(e, "reach"))
	want := n * (n - 1) / 2
	if got != want {
		t.Fatalf("reach closure = %d facts, want %d (a dropped far pair means the delta stopped early)", got, want)
	}
	// Spot-check the farthest pair specifically — the last thing a broken delta
	// would derive.
	if !slices.Contains(queryKeys(e, "reach"), "reach(a0,a7)") {
		t.Fatalf("missing reach(a0,a7) — the full-length transitive pair was not derived")
	}
}
