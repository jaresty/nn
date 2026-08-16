package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// shufEquivFixture builds a deterministic notebook + a single-paragraph input
// file. With exactly one sample unit, rand.Perm(1) is deterministic, so shuf's
// full stdout is stable and can be pinned exactly.
func shufEquivFixture(t *testing.T) (func(...string) (string, error), string) {
	t.Helper()
	nbDir, execute := setupNotebook(t)
	notes := []struct{ id, title, body string }{
		{"20260101000000-3001", "Auth token validation", "handleAuth validates the session token before routing."},
		{"20260101000000-3002", "Session middleware", "The session middleware wires handleAuth into each request."},
	}
	for _, n := range notes {
		doc := fmt.Sprintf("---\nid: %s\ntitle: %s\ntype: concept\nstatus: reviewed\n---\n%s\n", n.id, n.title, n.body)
		if err := os.WriteFile(filepath.Join(nbDir, n.id+".md"), []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(f, []byte("handleAuth validates the session token before routing the request.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return execute, f
}

// TestShufOutputEquivalence pins nn shuf's complete stdout for a single-unit
// input (deterministic sampling). It is the regression guard for the per-sample
// ranking hoist:
//
//	∀ input i: stdout_after(i) == stdout_before(i)
func TestShufOutputEquivalence(t *testing.T) {
	execute, f := shufEquivFixture(t)
	out, err := execute("shuf", f, "--count", "1", "--unit", "paragraphs")
	if err != nil {
		t.Fatalf("nn shuf: %v", err)
	}
	if out != wantShufOutput() {
		t.Fatalf("shuf output changed.\n--- got ---\n%q\n--- want ---\n%q", out, wantShufOutput())
	}
}

// wantShufOutput is the golden stdout captured from the pre-hoist nn shuf for the
// shufEquivFixture. Any change to sample printing, ranking, ordering, or labels
// breaks the equality assertion.
func wantShufOutput() string {
	return "---\n" +
		"handleAuth validates the session token before routing the request.\n" +
		"\n## Related notes\n" +
		"- [[20260101000000-3001|Auth token validation]]\n" +
		"- [[20260101000000-3002|Session middleware]]\n"
}
