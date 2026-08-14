package rules

import (
	"sort"
	"testing"
)

// factKeys returns the sorted string keys of all facts matching pred.
func queryKeys(e *Engine, pred string) []string {
	var out []string
	for _, f := range e.Query(pred) {
		out = append(out, f.Key())
	}
	sort.Strings(out)
	return out
}

func TestEngine_TransitiveClosure(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "link", Args: []string{"a", "b", "governs"}})
	e.AddFact(Fact{Pred: "link", Args: []string{"b", "c", "refines"}})
	e.AddFact(Fact{Pred: "link", Args: []string{"c", "d", "refines"}})

	rules := mustParse(t,
		`tgov(X, Y) :- link(X, Y, "governs").`,
		`tgov(X, Z) :- tgov(X, Y), link(Y, Z, "refines").`,
	)
	e.AddRules(rules)
	if err := e.Eval(); err != nil {
		t.Fatalf("Eval: %v", err)
	}

	got := queryKeys(e, "tgov")
	want := []string{"tgov(a,b)", "tgov(a,c)", "tgov(a,d)"}
	if !equalSlices(got, want) {
		t.Fatalf("tgov = %v, want %v", got, want)
	}
}

func TestEngine_StratifiedNegation(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "note", Args: []string{"a", "concept", "reviewed"}})
	e.AddFact(Fact{Pred: "note", Args: []string{"b", "concept", "draft"}})

	rules := mustParse(t,
		`reviewed(N) :- note(N, _, "reviewed").`,
		`unreviewed(N) :- note(N, _, _), !reviewed(N).`,
	)
	e.AddRules(rules)
	if err := e.Eval(); err != nil {
		t.Fatalf("Eval: %v", err)
	}

	got := queryKeys(e, "unreviewed")
	want := []string{"unreviewed(b)"}
	if !equalSlices(got, want) {
		t.Fatalf("unreviewed = %v, want %v", got, want)
	}
}

func TestEngine_RejectsNonStratifiable(t *testing.T) {
	e := NewEngine()
	// p depends on !p directly — negation through self-recursion.
	rules := mustParse(t,
		`p(X) :- q(X), !p(X).`,
	)
	e.AddFact(Fact{Pred: "q", Args: []string{"x"}})
	e.AddRules(rules)
	if err := e.Eval(); err == nil {
		t.Fatal("expected non-stratifiable ruleset to be rejected, got nil error")
	}
}

// Property: a negative cycle spanning MORE THAN ONE hop must also be rejected,
// not just direct self-negation. p depends on q; q depends on !p.
func TestEngine_RejectsMultiHopNegativeCycle(t *testing.T) {
	e := NewEngine()
	rules := mustParse(t,
		`p(X) :- r(X), q(X).`,
		`q(X) :- r(X), !p(X).`,
	)
	e.AddFact(Fact{Pred: "r", Args: []string{"x"}})
	e.AddRules(rules)
	if err := e.Eval(); err == nil {
		t.Fatal("expected multi-hop negative cycle to be rejected, got nil error")
	}
}

// Property: a POSITIVE recursive cycle (ordinary transitive closure) must be
// ACCEPTED — stratification only forbids negative cycles.
func TestEngine_AcceptsPositiveRecursion(t *testing.T) {
	e := NewEngine()
	rules := mustParse(t,
		`reach(X, Y) :- edge(X, Y).`,
		`reach(X, Z) :- reach(X, Y), edge(Y, Z).`,
	)
	e.AddFact(Fact{Pred: "edge", Args: []string{"a", "b"}})
	e.AddRules(rules)
	if err := e.Eval(); err != nil {
		t.Fatalf("positive recursion must be accepted, got %v", err)
	}
}

// ── test helpers ──────────────────────────────────────────────────────────────

func mustParse(t *testing.T, lines ...string) []Rule {
	t.Helper()
	var rs []Rule
	for _, l := range lines {
		r, err := ParseRule(l)
		if err != nil {
			t.Fatalf("ParseRule(%q): %v", l, err)
		}
		rs = append(rs, r)
	}
	return rs
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
