package cmd

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

// noteGoFile is the path to internal/note/note.go relative to the test package dir.
const noteGoFile = "../../../internal/note/note.go"

func TestAstGoFile(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("ast", noteGoFile)
	if err != nil {
		t.Fatalf("nn ast: %v", err)
	}
	if !strings.Contains(out, "language: go") {
		t.Errorf("ast output missing 'language: go':\n%s", out)
	}
	// Should show the Note type or Parse function.
	if !strings.Contains(out, "Note") && !strings.Contains(out, "Parse") {
		t.Errorf("ast output missing expected symbols:\n%s", out)
	}
}

func TestAstJSON(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("ast", "--json", noteGoFile)
	if err != nil {
		t.Fatalf("nn ast --json: %v", err)
	}
	var symbols []struct {
		Kind      string `json:"kind"`
		Name      string `json:"name"`
		Signature string `json:"signature"`
		Line      int    `json:"line"`
	}
	if err := json.Unmarshal([]byte(out), &symbols); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(symbols) == 0 {
		t.Error("expected at least one symbol in JSON output")
	}
}

func TestAstRefs(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("ast", noteGoFile, "--refs", "--root", "../../../")
	if err != nil {
		t.Fatalf("nn ast --refs: %v", err)
	}
	if !strings.Contains(out, "references to") {
		t.Errorf("ast --refs missing 'references to' output:\n%s", out)
	}
}

const bm25GoFile = "../../../internal/note/bm25.go"

func TestAstRefsSymbolFilter(t *testing.T) {
	_, execute := setupNotebook(t)

	// Only request refs for BM25Scores — a symbol defined in bm25.go with known uses.
	out, err := execute("ast", bm25GoFile, "--refs", "--symbol", "BM25Scores", "--root", "../../../internal/note")
	if err != nil {
		t.Fatalf("nn ast --refs --symbol: %v", err)
	}
	if !strings.Contains(out, "BM25Scores") {
		t.Errorf("expected BM25Scores in output:\n%s", out)
	}
	// Should NOT contain refs sections for other symbols.
	if strings.Contains(out, `references to "tokenize"`) {
		t.Errorf("--symbol filter should suppress other symbols:\n%s", out)
	}
}

func TestAstRefsEmptySectionsSuppressed(t *testing.T) {
	_, execute := setupNotebook(t)

	// Scan a tiny root that won't contain matches for most symbols.
	out, err := execute("ast", bm25GoFile, "--refs", "--symbol", "BM25Scores", "--root", "../../../internal/note")
	if err != nil {
		t.Fatalf("nn ast --refs: %v", err)
	}
	// Verify that bm25.go matches are found (non-empty output for BM25Scores).
	if !strings.Contains(out, "bm25") {
		t.Errorf("expected bm25.go match in output:\n%s", out)
	}
	// Empty sections for other symbols must be suppressed.
	// Since we only asked for BM25Scores, no other "references to" blocks should appear.
	if strings.Contains(out, `references to "tokenize"`) {
		t.Errorf("empty/filtered sections must be suppressed:\n%s", out)
	}
}

func TestAstRefsSummaryLine(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("ast", bm25GoFile, "--refs", "--symbol", "BM25Scores", "--root", "../../../internal/note")
	if err != nil {
		t.Fatalf("nn ast --refs: %v", err)
	}
	if !strings.Contains(out, "files scanned") {
		t.Errorf("expected summary line with 'files scanned' in output:\n%s", out)
	}
}

// TestAstRefsDefaultRootIsFileDir checks that --root defaults to the directory of the
// analyzed file, not ".". bm25.go is in internal/note/ so --refs without --root should
// find BM25Scores references in that same directory.
func TestAstRefsDefaultRootIsFileDir(t *testing.T) {
	_, execute := setupNotebook(t)

	// No --root flag — should default to internal/note (dir of bm25.go), not cwd.
	out, err := execute("ast", bm25GoFile, "--refs", "--symbol", "BM25Scores")
	if err != nil {
		t.Fatalf("nn ast --refs (no --root): %v", err)
	}
	// Must find bm25.go itself (in-file reference).
	if !strings.Contains(out, "bm25") {
		t.Errorf("expected bm25.go matches without explicit --root:\n%s", out)
	}
	// Summary must show at least 1 file scanned (not 0).
	if regexp.MustCompile(`\b0 files scanned\b`).MatchString(out) {
		t.Errorf("default root scanned 0 files — not defaulting to file dir:\n%s", out)
	}
}

// TestAstRefsRelativeRootDotDot checks that a relative --root with .. traversal works
// (regression guard for the filepath.Base("..")  == ".." skip bug).
func TestAstRefsRelativeRootDotDot(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("ast", bm25GoFile, "--refs", "--symbol", "BM25Scores", "--root", "../../../internal/note")
	if err != nil {
		t.Fatalf("nn ast --refs --root ../../../internal/note: %v", err)
	}
	if !strings.Contains(out, "bm25") {
		t.Errorf("expected matches with relative .. root:\n%s", out)
	}
}

func TestAstFooter(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("ast", noteGoFile)
	if err != nil {
		t.Fatalf("nn ast: %v", err)
	}
	if !strings.Contains(out, "## Related notes") {
		t.Errorf("ast output missing '## Related notes' section:\n%s", out)
	}
}

func TestAstJSONNoFooter(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("ast", "--json", noteGoFile)
	if err != nil {
		t.Fatalf("nn ast --json: %v", err)
	}
	if strings.Contains(out, "Resolve each unread related note") {
		t.Errorf("ast --json must not contain resolution footer:\n%s", out)
	}
}

// TestAstBM25UsesSymbolBody verifies that BM25 note matching uses the full symbol
// body, not just the symbol name. A note whose title matches a term in the function
// body (but not the function name) must surface in ## Related notes.
func TestAstBM25UsesSymbolBody(t *testing.T) {
	notebook, execute := setupNotebook(t)

	// Create a note whose title contains a term present only in the function body.
	if _, err := execute("new", "--title", "distinctivebodytermqrs usage pattern", "--type", "observation", "--no-edit"); err != nil {
		t.Fatalf("nn new: %v", err)
	}

	// Write a Go file with a function whose body contains the distinctive term.
	goFile := notebook + "/subject.go"
	content := `package main

func MyHandler() {
	// distinctivebodytermqrs
	_ = 42
}
`
	if err := os.WriteFile(goFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("ast", goFile)
	if err != nil {
		t.Fatalf("nn ast: %v", err)
	}
	if !strings.Contains(out, "distinctivebodytermqrs") {
		t.Errorf("FAIL: TestAstBM25UsesSymbolBody: expected note with body term 'distinctivebodytermqrs' to appear in Related notes, but got:\n%s", out)
	}
}

func TestAstUnknownLanguage(t *testing.T) {
	_, execute := setupNotebook(t)

	// A .xyz file won't be recognized.
	_, err := execute("ast", "nonexistent.xyz")
	if err == nil {
		t.Error("nn ast on unknown language should return error")
	}
}

func TestNewFromFile(t *testing.T) {
	_, cfgFile := setupNotebookWithCfg(t)

	root := NewRootCmdForTest(cfgFile)
	root.SetArgs([]string{"new", "--title", "From File Note", "--type", "concept", "--from-file", noteGoFile})
	var stdout strings.Builder
	root.SetOut(&stdout)

	if err := root.Execute(); err != nil {
		t.Fatalf("nn new --from-file: %v", err)
	}
	if !strings.Contains(stdout.String(), "created") {
		t.Errorf("expected 'created' in output: %q", stdout.String())
	}
}
