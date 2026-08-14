package rules

import (
	"strings"
	"testing"
)

// A scoped fence restricts its rules to notes with the given representation. The
// rule flags any note as bad(); with scope=taxonomy it must flag only the
// taxonomy note, not the axiom note.
func TestScope_RestrictsToRepresentation(t *testing.T) {
	body := "```nn-rule scope=taxonomy\n" +
		"flagged(N) :- note(N, _, _).\n" +
		"```\n"
	rules, warns := ExtractFenceRules("origin", body)
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}

	e := NewEngine()
	e.AddFact(Fact{Pred: "note", Args: []string{"tax", "concept", "reviewed"}})
	e.AddFact(Fact{Pred: "note", Args: []string{"ax", "concept", "reviewed"}})
	e.AddFact(Fact{Pred: "representation", Args: []string{"tax", "taxonomy"}})
	e.AddFact(Fact{Pred: "representation", Args: []string{"ax", "axiom"}})
	e.AddRules(rules)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}
	got := queryKeys(e, "flagged")
	want := []string{"flagged(tax)"} // only the taxonomy note is in scope
	if !equalSlices(got, want) {
		t.Fatalf("flagged = %v, want %v (scope must exclude the axiom note)", got, want)
	}
}

// An unscoped fence is unchanged: the same rule flags every note.
func TestScope_UnscopedUnchanged(t *testing.T) {
	body := "```nn-rule\n" +
		"flagged(N) :- note(N, _, _).\n" +
		"```\n"
	rules, _ := ExtractFenceRules("origin", body)

	e := NewEngine()
	e.AddFact(Fact{Pred: "note", Args: []string{"tax", "concept", "reviewed"}})
	e.AddFact(Fact{Pred: "note", Args: []string{"ax", "concept", "reviewed"}})
	e.AddFact(Fact{Pred: "representation", Args: []string{"tax", "taxonomy"}})
	e.AddRules(rules)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}
	if got := queryKeys(e, "flagged"); !equalSlices(got, []string{"flagged(ax)", "flagged(tax)"}) {
		t.Fatalf("unscoped flagged = %v, want both notes", got)
	}
}

// A head with no variable argument cannot be scoped — the rule loads unscoped
// and a provenance warning is emitted rather than dropping it.
func TestScope_NoVariableHeadWarnsAndLoads(t *testing.T) {
	// Head has an argument but no variable, so there is no subject to scope.
	body := "```nn-rule scope=taxonomy\n" +
		"marker(a).\n" +
		"```\n"
	rules, warns := ExtractFenceRules("origin", body)
	if len(rules) != 1 {
		t.Fatalf("expected the rule to still load, got %d rules", len(rules))
	}
	if len(warns) == 0 || !strings.Contains(warns[0], "scope=taxonomy ignored") {
		t.Fatalf("expected a scope-ignored warning, got %v", warns)
	}
}
