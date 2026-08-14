package rules

import (
	"testing"
	"time"
)

// P3: count aggregation binds a variable to the number of distinct values of a
// grouped variable. Surface form:
//   outdeg(N, K) :- count(T : link(N, T, _)) = K.
// meaning "K is the number of distinct T such that link(N, T, _)", grouped by
// the head's non-aggregated variables (here N).
func TestAggregation_CountDistinctOutbound(t *testing.T) {
	e := NewEngine()
	// n has two distinct outbound targets; solo has one.
	e.AddFact(Fact{Pred: "link", Args: []string{"n", "a", "refines"}})
	e.AddFact(Fact{Pred: "link", Args: []string{"n", "b", "refines"}})
	e.AddFact(Fact{Pred: "link", Args: []string{"solo", "x", "refines"}})

	rs := mustParse(t,
		`outdeg(N, K) :- count(T : link(N, T, _)) = K.`,
		`well_connected(N) :- outdeg(N, K), K = "2".`,
	)
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}

	// n has outdeg 2, solo has outdeg 1.
	if got := queryKeys(e, "outdeg"); !equalSlices(got, []string{"outdeg(n,2)", "outdeg(solo,1)"}) {
		t.Fatalf("outdeg = %v, want [outdeg(n,2) outdeg(solo,1)]", got)
	}
	// well_connected fires only for n (2+ distinct outbound) — the exact v1 gap.
	if got := queryKeys(e, "well_connected"); !equalSlices(got, []string{"well_connected(n)"}) {
		t.Fatalf("well_connected = %v, want [well_connected(n)]", got)
	}
}

// P3b: an aggregate rule (which has an empty ordinary Body) must NOT emit a
// spurious all-empty-args fact via the ordinary fixpoint pass.
func TestAggregation_NoSpuriousEmptyGroup(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "link", Args: []string{"n", "a", "refines"}})

	rs := mustParse(t,
		`outdeg(N, K) :- count(T : link(N, T, _)) = K.`,
	)
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}
	for _, f := range e.Query("outdeg") {
		if f.Args[0] == "" {
			t.Fatalf("aggregate emitted a spurious empty-group fact: %q", f.Key())
		}
	}
}

// P6: a malformed aggregate — result variable not present in the head — is a
// parse error, not a silent misparse.
func TestAggregation_ResultVarNotInHeadRejected(t *testing.T) {
	// K is the result var but the head is outdeg(N, Z) — K is absent.
	_, err := ParseRule(`outdeg(N, Z) :- count(T : link(N, T, _)) = K.`)
	if err == nil {
		t.Fatal("expected parse error when aggregate result variable is absent from head, got nil")
	}
}

// P4: an aggregate whose source predicate (transitively) depends on the
// aggregate's own head is not stratifiable — counting a relation that is still
// growing as a result of the count is ill-defined. Eval must reject it QUICKLY
// with a non-nil error, not hang. The guard bounds Eval on a deadline so the
// witness is an explicit error, not a timeout.
func TestAggregation_SelfReferentialRejected(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "seed", Args: []string{"a", "b"}})
	// grew(X,Y) depends on cnt (the aggregate head), and cnt counts grew —
	// a cycle through an aggregate.
	rs := mustParse(t,
		`cnt(X, K) :- count(Y : grew(X, Y)) = K.`,
		`grew(X, Y) :- seed(X, Y).`,
		`grew(X, Y) :- cnt(X, Y).`,
	)
	e.AddRules(rs)

	done := make(chan error, 1)
	go func() { done <- e.Eval() }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected non-stratifiable error for aggregate depending on its own output, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Eval did not return within 2s — self-referential aggregate was not rejected (would hang)")
	}
}

// P3: count over distinct values only — duplicate identical rows count once.
func TestAggregation_CountsDistinct(t *testing.T) {
	e := NewEngine()
	// Two links to the SAME target a, plus one to b ⇒ 2 distinct targets.
	e.AddFact(Fact{Pred: "link", Args: []string{"n", "a", "refines"}})
	e.AddFact(Fact{Pred: "link", Args: []string{"n", "a", "extends"}}) // same target a
	e.AddFact(Fact{Pred: "link", Args: []string{"n", "b", "refines"}})

	rs := mustParse(t,
		`outdeg(N, K) :- count(T : link(N, T, _)) = K.`,
	)
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}
	if got := queryKeys(e, "outdeg"); !equalSlices(got, []string{"outdeg(n,2)"}) {
		t.Fatalf("distinct count wrong: outdeg = %v, want [outdeg(n,2)]", got)
	}
}
