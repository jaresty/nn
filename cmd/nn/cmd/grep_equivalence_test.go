package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// grepEquivNotebook builds a small deterministic notebook + multi-match code file
// used to pin nn grep's exact output. It seeds notes whose vocabulary overlaps the
// match context so the BM25 annotate path produces ranked related notes.
func grepEquivNotebook(t *testing.T) (func(...string) (string, error), string) {
	t.Helper()
	nbDir, execute := setupNotebook(t)

	notes := []struct{ id, title, body string }{
		{"20260101000000-1001", "Auth token validation", "handleAuth validates the session token before routing."},
		{"20260101000000-1002", "Session middleware", "The session middleware wires handleAuth into each request."},
		{"20260101000000-1003", "Request routing", "Routing dispatches to handleAuth for protected paths."},
	}
	for _, n := range notes {
		doc := fmt.Sprintf("---\nid: %s\ntitle: %s\ntype: concept\nstatus: reviewed\n---\n%s\n", n.id, n.title, n.body)
		if err := os.WriteFile(filepath.Join(nbDir, n.id+".md"), []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	dir := t.TempDir()
	var body string
	for i := 0; i < 5; i++ {
		body += fmt.Sprintf("// session token routing middleware %d\nfunc handleAuth%d() { validateToken() }\n\n", i, i)
	}
	f := filepath.Join(dir, "server.go")
	if err := os.WriteFile(f, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return execute, f
}

// TestGrepOutputEquivalence pins nn grep's complete stdout for a fixed input.
// It is the regression guard for the optimization property:
//
//	∀ input i: stdout_after(i) == stdout_before(i)
//
// Any change to grep's output — including an optimization that alters per-match
// ranking, ordering, labels, or context — makes this test fail. The expected
// value is captured from the pre-optimization implementation.
func TestGrepOutputEquivalence(t *testing.T) {
	execute, f := grepEquivNotebook(t)
	out, err := execute("grep", "handleAuth", f, "--max-matches", "0", "--context", "1")
	if err != nil {
		t.Fatalf("nn grep: %v", err)
	}
	if out != wantGrepOutput(f) {
		t.Fatalf("grep output changed.\n--- got ---\n%q\n--- want ---\n%q", out, wantGrepOutput(f))
	}
}

// wantGrepOutput is the golden stdout captured from the pre-optimization grep for
// the grepEquivNotebook fixture at --context 1 --max-matches 0. Parameterized by
// the temp file path. Any change to grep's match printing, context, ranking,
// ordering, or labels breaks the equality assertion in TestGrepOutputEquivalence.
func wantGrepOutput(f string) string {
	block := func(ctxLine, matchLine, gap int) string {
		return fmt.Sprintf("%d:// session token routing middleware %d\n", ctxLine, gap) +
			fmt.Sprintf("%d:func handleAuth%d() { validateToken() }\n", matchLine, gap) +
			fmt.Sprintf("%d:\n", matchLine+1) +
			"  → [[20260101000000-1002|Session middleware]] [possibly relevant]\n" +
			"  → [[20260101000000-1003|Request routing]] [possibly relevant]\n"
	}
	return "==> " + f + " <==\n" +
		block(1, 2, 0) +
		block(4, 5, 1) +
		block(7, 8, 2) +
		block(10, 11, 3) +
		block(13, 14, 4) +
		"Resolve each unread related note — run `nn show <id>` to open, or write `skip-related: <id> [<id> ...] — <reason>` to dismiss. Notes marked [read] have already been loaded this session and do not require action.\n"
}
