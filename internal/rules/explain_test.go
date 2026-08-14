package rules

import (
	"strings"
	"testing"
)

// Property: after evaluation, the engine can explain how a derived fact was
// produced — the rule that fired and the premise facts it consumed — recursively
// down to base facts.

func TestExplain_DerivationPath(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "link", Args: []string{"a", "b", "governs"}})
	e.AddFact(Fact{Pred: "link", Args: []string{"b", "c", "refines"}})

	rs := mustParse(t,
		`tgov(X, Y) :- link(X, Y, "governs").`,
		`tgov(X, Z) :- tgov(X, Y), link(Y, Z, "refines").`,
	)
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}

	// Explain the transitively-derived fact tgov(a,c).
	steps, ok := e.Explain(Fact{Pred: "tgov", Args: []string{"a", "c"}})
	if !ok {
		t.Fatal("Explain returned ok=false for a derived fact")
	}
	text := strings.Join(steps, "\n")

	// The explanation must reference the target fact, the intermediate derived
	// fact tgov(a,b), and the base fact it rests on.
	for _, want := range []string{"tgov(a,c)", "tgov(a,b)", "link(b,c,refines)", "link(a,b,governs)"} {
		if !strings.Contains(text, want) {
			t.Errorf("explanation missing %q; got:\n%s", want, text)
		}
	}
}

func TestExplain_BaseFactHasNoDerivation(t *testing.T) {
	e := NewEngine()
	e.AddFact(Fact{Pred: "note", Args: []string{"a", "concept", "draft"}})
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}
	steps, ok := e.Explain(Fact{Pred: "note", Args: []string{"a", "concept", "draft"}})
	if !ok {
		t.Fatal("Explain should succeed for a present base fact")
	}
	text := strings.Join(steps, "\n")
	if !strings.Contains(text, "base fact") {
		t.Errorf("base fact should be labelled as such; got:\n%s", text)
	}
}

func TestExplain_UnknownFact(t *testing.T) {
	e := NewEngine()
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}
	if _, ok := e.Explain(Fact{Pred: "nope", Args: []string{"x"}}); ok {
		t.Fatal("Explain should return ok=false for a fact that was never derived")
	}
}
