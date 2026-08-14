package rules

import "testing"

// P1: comparison literals != and = filter bindings.
func TestComparison_Inequality(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "link", Args: []string{"n", "a", "refines"}})
	e.AddFact(Fact{Pred: "link", Args: []string{"n", "b", "refines"}})

	// two_distinct(N) holds iff N has two DIFFERENT outbound targets.
	rs := mustParse(t,
		`two_distinct(N) :- link(N, A, _), link(N, B, _), A != B.`,
	)
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}
	got := queryKeys(e, "two_distinct")
	want := []string{"two_distinct(n)"}
	if !equalSlices(got, want) {
		t.Fatalf("two_distinct = %v, want %v", got, want)
	}
}

// P1: with only ONE outbound link, != must NOT match (A and B unify to the same
// target, so A != B fails) — this is the exact v1 gap the feature closes.
func TestComparison_InequalitySingleLinkExcluded(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "link", Args: []string{"solo", "x", "refines"}})

	rs := mustParse(t,
		`two_distinct(N) :- link(N, A, _), link(N, B, _), A != B.`,
	)
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}
	if got := queryKeys(e, "two_distinct"); len(got) != 0 {
		t.Fatalf("single-link note must not match two_distinct, got %v", got)
	}
}

// P2: a comparison whose operand is never bound by a positive body literal is
// a rule error (unsafe/unrange-restricted), not a silent no-op.
func TestComparison_UnboundOperandRejected(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "p", Args: []string{"a"}})
	// Z appears only in the comparison, never bound by a positive literal.
	rs := mustParse(t,
		`bad(X) :- p(X), X != Z.`,
	)
	e.AddRules(rs)
	if err := e.Eval(); err == nil {
		t.Fatal("expected error for comparison with an unbound operand, got nil")
	}
}

// P1: equality keeps only solutions where the two vars are equal.
func TestComparison_Equality(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "pair", Args: []string{"a", "a"}})
	e.AddFact(Fact{Pred: "pair", Args: []string{"a", "b"}})

	rs := mustParse(t,
		`same(X, Y) :- pair(X, Y), X = Y.`,
	)
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}
	got := queryKeys(e, "same")
	want := []string{"same(a,a)"}
	if !equalSlices(got, want) {
		t.Fatalf("same = %v, want %v", got, want)
	}
}
