package rules

import "testing"

// Ordering comparisons (<, <=, >, >=) compare operands numerically. A rule that
// keeps only bindings whose count is below a threshold must fire on the small
// values and reject the large ones — and it must be numeric, not lexical
// ("10" < "2" lexically, but 10 < 2 is false numerically).
func TestOrdering_NumericLessThan(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "k", Args: []string{"a", "2"}})
	e.AddFact(Fact{Pred: "k", Args: []string{"b", "10"}})

	rs := mustParse(t, `small(N) :- k(N, V), V < 3.`)
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}
	got := queryKeys(e, "small")
	want := []string{"small(a)"} // 2 < 3 true; 10 < 3 false
	if !equalSlices(got, want) {
		t.Fatalf("small = %v, want %v (numeric, not lexical)", got, want)
	}
}

// >, <=, >= round out the operator set.
func TestOrdering_AllOperators(t *testing.T) {
	cases := []struct {
		op   string
		want []string // which of a(=2), b(=10) satisfy V <op> 3
	}{
		{"<", []string{"m(a)"}},
		{"<=", []string{"m(a)"}},
		{">", []string{"m(b)"}},
		{">=", []string{"m(b)"}},
	}
	for _, c := range cases {
		t.Run(c.op, func(t *testing.T) {
			e := NewEngine()
			e.AddFact(Fact{Pred: "k", Args: []string{"a", "2"}})
			e.AddFact(Fact{Pred: "k", Args: []string{"b", "10"}})
			rs := mustParse(t, "m(N) :- k(N, V), V "+c.op+" 3.")
			e.AddRules(rs)
			if err := e.Eval(); err != nil {
				t.Fatalf("eval: %v", err)
			}
			if got := queryKeys(e, "m"); !equalSlices(got, c.want) {
				t.Fatalf("V %s 3: m = %v, want %v", c.op, got, c.want)
			}
		})
	}
}

// A non-numeric operand makes an ordering comparison fail (not error).
func TestOrdering_NonNumericFails(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "k", Args: []string{"a", "hello"}})

	rs := mustParse(t, `bad(N) :- k(N, V), V < 3.`)
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval must not error on non-numeric operand: %v", err)
	}
	if got := queryKeys(e, "bad"); len(got) != 0 {
		t.Fatalf("non-numeric operand must not satisfy ordering, got %v", got)
	}
}
