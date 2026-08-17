package cmd

import (
	"strings"
	"testing"
)

// property [3]: nn rules list names the built-in queryable derived predicates so
// they are discoverable, not just a rule count.
func TestRulesListNamesBuiltinPredicates(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("rules", "list")
	if err != nil {
		t.Fatalf("nn rules list: %v", err)
	}
	for _, pred := range []string{"blocked", "done", "reachable", "transitively_governs"} {
		if !strings.Contains(out, pred) {
			t.Errorf("nn rules list does not name built-in derived predicate %q; got:\n%s", pred, out)
		}
	}
}

// property [4]: the nn-cli-reference protocol documents the queryable derived
// predicates, including the new task-dependency ones.
func TestProtocolDocumentsDerivedPredicates(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("nn show virtual-nn-cli-reference: %v", err)
	}
	for _, s := range []string{"queryable derived predicates", "blocked(X)", "done(X)"} {
		if !strings.Contains(out, s) {
			t.Errorf("protocol does not document %q; got:\n%s", s, out)
		}
	}
}
