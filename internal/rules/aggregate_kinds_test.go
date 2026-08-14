package rules

import "testing"

// sum folds a multiset: two facts with the same value both count, so the sum of
// {2, 2, 3} is 7 (a distinct-set fold would wrongly give 5).
func TestAggregate_SumIsMultiset(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "v", Args: []string{"g", "x1", "2"}})
	e.AddFact(Fact{Pred: "v", Args: []string{"g", "x2", "2"}})
	e.AddFact(Fact{Pred: "v", Args: []string{"g", "x3", "3"}})

	rs := mustParse(t, `total(G, S) :- sum(N : v(G, _, N)) = S.`)
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}
	got := queryKeys(e, "total")
	want := []string{"total(g,7)"} // 2 + 2 + 3
	if !equalSlices(got, want) {
		t.Fatalf("total = %v, want %v (sum must be a multiset)", got, want)
	}
}

func TestAggregate_MinMax(t *testing.T) {
	build := func() *Engine {
		e := NewEngine()
		e.AddFact(Fact{Pred: "v", Args: []string{"g", "x1", "5"}})
		e.AddFact(Fact{Pred: "v", Args: []string{"g", "x2", "2"}})
		e.AddFact(Fact{Pred: "v", Args: []string{"g", "x3", "9"}})
		return e
	}

	e := build()
	e.AddRules(mustParse(t, `lo(G, M) :- min(N : v(G, _, N)) = M.`))
	if err := e.Eval(); err != nil {
		t.Fatalf("eval min: %v", err)
	}
	if got := queryKeys(e, "lo"); !equalSlices(got, []string{"lo(g,2)"}) {
		t.Fatalf("min = %v, want [lo(g,2)]", got)
	}

	e = build()
	e.AddRules(mustParse(t, `hi(G, M) :- max(N : v(G, _, N)) = M.`))
	if err := e.Eval(); err != nil {
		t.Fatalf("eval max: %v", err)
	}
	if got := queryKeys(e, "hi"); !equalSlices(got, []string{"hi(g,9)"}) {
		t.Fatalf("max = %v, want [hi(g,9)]", got)
	}
}

// A group whose aggregated values are all non-numeric yields no head fact for
// sum/min/max (and does not error).
func TestAggregate_SumNonNumericOmitted(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "v", Args: []string{"g", "x1", "hello"}})

	rs := mustParse(t, `total(G, S) :- sum(N : v(G, _, N)) = S.`)
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}
	if got := queryKeys(e, "total"); len(got) != 0 {
		t.Fatalf("all-non-numeric group must yield no fact, got %v", got)
	}
}

// count semantics are unchanged: DISTINCT values, so {2, 2, 3} counts as 2.
func TestAggregate_CountStillDistinct(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "v", Args: []string{"g", "x1", "2"}})
	e.AddFact(Fact{Pred: "v", Args: []string{"g", "x2", "2"}})
	e.AddFact(Fact{Pred: "v", Args: []string{"g", "x3", "3"}})

	rs := mustParse(t, `c(G, K) :- count(N : v(G, _, N)) = K.`)
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}
	if got := queryKeys(e, "c"); !equalSlices(got, []string{"c(g,2)"}) {
		t.Fatalf("count = %v, want [c(g,2)] (distinct values)", got)
	}
}
