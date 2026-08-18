package cmd

import (
	"strings"
	"testing"
)

func newTaskNote(t *testing.T, execute func(...string) (string, error), content string) string {
	t.Helper()
	out, err := execute("new", "--title", "Task note", "--type", "observation", "--content", content, "--no-edit")
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	id := strings.TrimPrefix(strings.TrimSpace(out), "created ")
	return strings.Fields(id)[0]
}

// property [1]: one `nn todo done <id> <p1> <p2>` flips all matched checkboxes in
// a single call (one write), so parallel single-checkbox attempts are unneeded.
func TestTodoDoneMultiplePatterns(t *testing.T) {
	_, execute := setupNotebook(t)
	id := newTaskNote(t, execute, "- [ ] write tests\n- [ ] review PR\n- [ ] ship it")

	if _, err := execute("todo", "done", id, "write tests", "ship it"); err != nil {
		t.Fatalf("nn todo done multi: %v", err)
	}
	body := noteBody(t, execute, id)
	if !strings.Contains(body, "- [x] write tests") {
		t.Errorf("expected 'write tests' done; got:\n%s", body)
	}
	if !strings.Contains(body, "- [x] ship it") {
		t.Errorf("expected 'ship it' done; got:\n%s", body)
	}
	if !strings.Contains(body, "- [ ] review PR") {
		t.Errorf("expected 'review PR' still open; got:\n%s", body)
	}
}

// property [2]: all-or-nothing — if any pattern matches nothing, the call errors
// and no checkbox is flipped.
func TestTodoDoneMultiAllOrNothing(t *testing.T) {
	_, execute := setupNotebook(t)
	id := newTaskNote(t, execute, "- [ ] write tests\n- [ ] review PR")

	_, err := execute("todo", "done", id, "write tests", "nonexistent")
	if err == nil {
		t.Fatal("expected error when a pattern matches nothing")
	}
	body := noteBody(t, execute, id)
	if strings.Contains(body, "- [x]") {
		t.Errorf("all-or-nothing violated: a checkbox was flipped despite the error; got:\n%s", body)
	}
}

// property [5]: --resolution appends commentary to the flipped line(s).
func TestTodoDoneResolution(t *testing.T) {
	_, execute := setupNotebook(t)
	id := newTaskNote(t, execute, "- [ ] migrate schema\n- [ ] keep this open")

	if _, err := execute("todo", "done", id, "migrate schema", "--resolution", "done in PR #42"); err != nil {
		t.Fatalf("nn todo done --resolution: %v", err)
	}
	body := noteBody(t, execute, id)
	if !strings.Contains(body, "- [x] migrate schema — done in PR #42") {
		t.Errorf("expected resolution commentary appended; got:\n%s", body)
	}
	if !strings.Contains(body, "- [ ] keep this open") {
		t.Errorf("expected unrelated item untouched; got:\n%s", body)
	}
}

// property [1-conjunctive]: when all patterns are substrings of a single open
// checkbox line, that one line is flipped (conjunctive AND on one line), rather
// than requiring each pattern to match a distinct line.
func TestTodoDoneConjunctiveSingleLine(t *testing.T) {
	_, execute := setupNotebook(t)
	id := newTaskNote(t, execute, "- [ ] fix graph apply atomicity — needs targeted test\n- [ ] unrelated item")

	if _, err := execute("todo", "done", id, "atomicity", "targeted test"); err != nil {
		t.Fatalf("nn todo done conjunctive same-line: %v", err)
	}
	body := noteBody(t, execute, id)
	if !strings.Contains(body, "- [x] fix graph apply atomicity — needs targeted test") {
		t.Errorf("expected the single line matching both patterns to be flipped; got:\n%s", body)
	}
	if !strings.Contains(body, "- [ ] unrelated item") {
		t.Errorf("expected unrelated item untouched; got:\n%s", body)
	}
}

// property [4-precedence]: when patterns could match either a single all-containing
// line or separate distinct lines, the single-line conjunctive match wins (exactly
// one line flipped, not two).
func TestTodoDoneConjunctiveTakesPrecedence(t *testing.T) {
	_, execute := setupNotebook(t)
	// "alpha" and "beta" both appear on line 1; "beta" also appears alone on line 2.
	id := newTaskNote(t, execute, "- [ ] alpha and beta together\n- [ ] beta alone")

	if _, err := execute("todo", "done", id, "alpha", "beta"); err != nil {
		t.Fatalf("nn todo done precedence: %v", err)
	}
	body := noteBody(t, execute, id)
	if !strings.Contains(body, "- [x] alpha and beta together") {
		t.Errorf("expected the single all-containing line flipped; got:\n%s", body)
	}
	if !strings.Contains(body, "- [ ] beta alone") {
		t.Errorf("expected 'beta alone' untouched (conjunctive precedence); got:\n%s", body)
	}
}

// property [4]: reopen mirrors the multi-pattern behavior.
func TestTodoReopenMultiplePatterns(t *testing.T) {
	_, execute := setupNotebook(t)
	id := newTaskNote(t, execute, "- [x] a done\n- [x] b done\n- [x] c done")

	if _, err := execute("todo", "reopen", id, "a done", "c done"); err != nil {
		t.Fatalf("nn todo reopen multi: %v", err)
	}
	body := noteBody(t, execute, id)
	if !strings.Contains(body, "- [ ] a done") || !strings.Contains(body, "- [ ] c done") {
		t.Errorf("expected a/c reopened; got:\n%s", body)
	}
	if !strings.Contains(body, "- [x] b done") {
		t.Errorf("expected b still done; got:\n%s", body)
	}
}
