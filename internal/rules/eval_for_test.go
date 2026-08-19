package rules

import (
	"reflect"
	"testing"
)

type predicateEvaluator interface {
	EvalFor(...string) error
}

func requireEvalFor(t *testing.T, e *Engine, predicates ...string) {
	t.Helper()
	pe, ok := any(e).(predicateEvaluator)
	if !ok {
		t.Fatal("Engine must implement EvalFor for predicate-directed evaluation")
	}
	if err := pe.EvalFor(predicates...); err != nil {
		t.Fatalf("EvalFor(%v): %v", predicates, err)
	}
}

func TestEvalForEvaluatesOnlyDependencyClosure(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "seed", Args: []string{"x"}})
	e.AddRules(mustParse(t,
		`wanted(X) :- helper(X).`,
		`helper(X) :- seed(X).`,
		`unrelated(X) :- seed(X).`,
	))

	requireEvalFor(t, e, "wanted")
	if got := queryKeys(e, "wanted"); !reflect.DeepEqual(got, []string{"wanted(x)"}) {
		t.Fatalf("wanted = %v", got)
	}
	if got := e.Query("unrelated"); len(got) != 0 {
		t.Fatalf("unrelated rules evaluated: %v", got)
	}
}

func TestEvalForIncludesNegatedAndAggregateDependencies(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "item", Args: []string{"a"}})
	e.AddFact(Fact{Pred: "item", Args: []string{"b"}})
	e.AddFact(Fact{Pred: "open", Args: []string{"b"}})
	e.AddRules(mustParse(t,
		`closed(X) :- item(X), !open(X).`,
		`closed_count(N) :- count(X : closed(X)) = N.`,
		`summary(N) :- closed_count(N).`,
	))

	requireEvalFor(t, e, "summary")
	if got := queryKeys(e, "summary"); !reflect.DeepEqual(got, []string{"summary(1)"}) {
		t.Fatalf("summary = %v", got)
	}
}

func TestEvalForMatchesFullEvalForRequestedPredicates(t *testing.T) {
	program := mustParse(t,
		`reachable(X, Y) :- edge(X, Y).`,
		`reachable(X, Z) :- reachable(X, Y), edge(Y, Z).`,
		`has_open(X) :- open(X).`,
		`done(X) :- node(X), !has_open(X).`,
		`blocked(X) :- requires(X, T), !done(T).`,
		`blocked(X) :- requires(X, T), blocked(T).`,
	)
	build := func() *Engine {
		e := NewEngine()
		for _, f := range []Fact{
			{Pred: "node", Args: []string{"a"}},
			{Pred: "node", Args: []string{"b"}},
			{Pred: "node", Args: []string{"c"}},
			{Pred: "open", Args: []string{"c"}},
			{Pred: "requires", Args: []string{"a", "b"}},
			{Pred: "requires", Args: []string{"b", "c"}},
			{Pred: "edge", Args: []string{"a", "b"}},
			{Pred: "edge", Args: []string{"b", "c"}},
		} {
			e.AddFact(f)
		}
		e.AddRules(program)
		return e
	}

	full := build()
	if err := full.Eval(); err != nil {
		t.Fatal(err)
	}
	directed := build()
	requireEvalFor(t, directed, "blocked")
	if got, want := queryKeys(directed, "blocked"), queryKeys(full, "blocked"); !reflect.DeepEqual(got, want) {
		t.Fatalf("EvalFor blocked=%v, Eval blocked=%v", got, want)
	}
	if got := directed.Query("reachable"); len(got) != 0 {
		t.Fatalf("EvalFor(blocked) derived unrelated reachable facts: %v", got)
	}
}
