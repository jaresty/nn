package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// chain: A --requires--> B --requires--> C(open). B has an open item of its own
// AND is transitively blocked; A's DIRECT requirement B is itself done-of-its-own
// -work? No — to isolate the TRANSITIVE case, B has no open item (direct-done)
// but requires C which is open. So:
//   - A requires B; B has an open item ("do A"/"do B" both actionable text)
//   - Direct-only isBlocked(A) would need B not-done; here B has no open items so
//     B is directly-done — only the transitive rule (B requires open C) blocks A.
func blockedChain() ([]*note.Note, map[string]*note.Note) {
	// A has its own open item and requires B.
	a := &note.Note{ID: "a", Title: "A", Type: note.TypeConcept, Body: "- [ ] task A",
		Links: []note.Link{{TargetID: "b", Type: "requires"}}}
	// B has NO open checkbox (directly done) but requires C.
	b := &note.Note{ID: "b", Title: "B", Type: note.TypeConcept, Body: "B has no open items.",
		Links: []note.Link{{TargetID: "c", Type: "requires"}}}
	// C is open — the true blocker.
	c := &note.Note{ID: "c", Title: "C", Type: note.TypeConcept, Body: "- [ ] task C"}
	notes := []*note.Note{a, b, c}
	byID := map[string]*note.Note{"a": a, "b": b, "c": c}
	return notes, byID
}

// property [1]: the default todo view excludes A, which is blocked ONLY
// transitively — its direct requirement B has no open items (directly done), but
// B requires the still-open C. The direct-only isBlocked would wrongly INCLUDE A.
func TestCollectOpenTodos_ExcludesTransitivelyBlocked(t *testing.T) {
	notes, byID := blockedChain()
	out := collectOpenTodos(notes, byID, todoListOptions{})

	if strings.Contains(out, "task A") {
		t.Errorf("transitively-blocked note A should be excluded from default todo view; got:\n%s", out)
	}
	if !strings.Contains(out, "task C") {
		t.Errorf("unblocked note C should be included; got:\n%s", out)
	}

	// showAll includes A regardless of blocking.
	all := collectOpenTodos(notes, byID, todoListOptions{showAll: true})
	if !strings.Contains(all, "task A") {
		t.Errorf("--all should include blocked note A; got:\n%s", all)
	}
}

// property [2]: with no requires edges, no note is block-excluded (and no engine
// eval is needed — a note with open items simply shows).
func TestCollectOpenTodos_NoRequiresNoExclusion(t *testing.T) {
	n := &note.Note{ID: "x", Title: "X", Type: note.TypeConcept, Body: "- [ ] standalone task"}
	notes := []*note.Note{n}
	byID := map[string]*note.Note{"x": n}
	out := collectOpenTodos(notes, byID, todoListOptions{})
	if !strings.Contains(out, "standalone task") {
		t.Errorf("note with no requires links must not be block-excluded; got:\n%s", out)
	}
}
