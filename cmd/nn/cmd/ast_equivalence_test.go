package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// astEquivFixture builds a deterministic notebook + a small Go file whose symbol
// bodies overlap the note vocabulary, so nn ast's per-symbol BM25 annotation
// path produces ranked related notes.
func astEquivFixture(t *testing.T) (func(...string) (string, error), string) {
	t.Helper()
	nbDir, execute := setupNotebook(t)
	notes := []struct{ id, title, body string }{
		{"20260101000000-2001", "Auth token validation", "handleAuth validates the session token before routing."},
		{"20260101000000-2002", "Session middleware", "The session middleware wires handleAuth into each request."},
	}
	for _, n := range notes {
		doc := fmt.Sprintf("---\nid: %s\ntitle: %s\ntype: concept\nstatus: reviewed\n---\n%s\n", n.id, n.title, n.body)
		if err := os.WriteFile(filepath.Join(nbDir, n.id+".md"), []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	src := "package server\n\n" +
		"// handleAuth validates the session token before routing\n" +
		"func handleAuth() { validateToken() }\n"
	f := filepath.Join(dir, "server.go")
	if err := os.WriteFile(f, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return execute, f
}

// TestAstOutputEquivalence pins nn ast's complete stdout for a fixed input. It is
// the regression guard for the per-symbol ranking hoist:
//
//	∀ input i: stdout_after(i) == stdout_before(i)
func TestAstOutputEquivalence(t *testing.T) {
	execute, f := astEquivFixture(t)
	out, err := execute("ast", f)
	if err != nil {
		t.Fatalf("nn ast: %v", err)
	}
	if out != wantAstOutput(f) {
		t.Fatalf("ast output changed.\n--- got ---\n%q\n--- want ---\n%q", out, wantAstOutput(f))
	}
}

// wantAstOutput is the golden stdout captured from the pre-hoist nn ast for the
// astEquivFixture, parameterized by the temp file path. Any change to symbol
// printing, ranking, ordering, or labels breaks the equality assertion.
func wantAstOutput(f string) string {
	return "file: " + f + "  language: go\n" +
		"imports: \n" +
		"func handleAuth() { validateToken() }\n" +
		"\n## Related notes\n" +
		"- [[20260101000000-2001|Auth token validation]] [likely relevant]\n" +
		"- [[20260101000000-2002|Session middleware]] [likely relevant]\n" +
		"Resolve each unread related note — run `nn show <id>` to open, or write `skip-related: <id> [<id> ...] — <reason>` to dismiss. Notes marked [read] have already been loaded this session and do not require action.\n"
}
