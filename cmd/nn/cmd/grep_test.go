package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Assertion: TestGrepCmd — nn grep <pattern> <path> outputs file:line:content for each match.
func TestGrepCmd(t *testing.T) {
	_, execute := setupNotebook(t)

	dir := t.TempDir()
	f := filepath.Join(dir, "main.go")
	if err := os.WriteFile(f, []byte("package main\n\nfunc handleAuth() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "handleAuth", f)
	if err != nil {
		t.Fatalf("nn grep: %v", err)
	}
	if !strings.Contains(out, "handleAuth") {
		t.Errorf("expected match 'handleAuth' in output; got:\n%s", out)
	}
	if !strings.Contains(out, f+":") {
		t.Errorf("expected file path %q in output; got:\n%s", f, out)
	}
}

// Assertion: TestGrepCmdAnnotatesWithNotes — nn grep annotates matches with related nn notes.
func TestGrepCmdAnnotatesWithNotes(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// Create a note whose content overlaps the search context.
	noteFile := filepath.Join(nbDir, "20260101000000-0001.md")
	if err := os.WriteFile(noteFile, []byte("---\nid: 20260101000000-0001\ntitle: Auth flow design\ntype: concept\nstatus: draft\n---\nhandleAuth is responsible for token validation.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	f := filepath.Join(dir, "server.go")
	if err := os.WriteFile(f, []byte("package main\n\nfunc handleAuth() {\n\t// validate token\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "handleAuth", f)
	if err != nil {
		t.Fatalf("nn grep with notes: %v", err)
	}
	if !strings.Contains(out, "Auth flow design") {
		t.Errorf("expected related note title 'Auth flow design' in output; got:\n%s", out)
	}
}

// Assertion: TestGrepCmdRelatedNotesLabel — nn grep appends score-derived label per related note.
func TestGrepCmdRelatedNotesLabel(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	noteFile := filepath.Join(nbDir, "20260101000000-0003.md")
	if err := os.WriteFile(noteFile, []byte("---\nid: 20260101000000-0003\ntitle: Auth flow design\ntype: concept\nstatus: reviewed\n---\nhandleAuth is responsible for token validation.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	f := filepath.Join(dir, "server.go")
	if err := os.WriteFile(f, []byte("package main\n\nfunc handleAuth() {\n\t// validate token\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "handleAuth", f)
	if err != nil {
		t.Fatalf("nn grep: %v", err)
	}
	if !strings.Contains(out, "[likely relevant]") && !strings.Contains(out, "[possibly relevant]") {
		t.Errorf("expected score-derived label ([likely relevant] or [possibly relevant]) in grep output; got:\n%s", out)
	}
}

// Assertion: TestGrepCmdRelatedNotesGateHeader — nn grep footer uses gate language.
func TestGrepCmdRelatedNotesGateHeader(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	noteFile := filepath.Join(nbDir, "20260101000000-0004.md")
	if err := os.WriteFile(noteFile, []byte("---\nid: 20260101000000-0004\ntitle: Auth flow design\ntype: concept\nstatus: reviewed\n---\nhandleAuth is responsible for token validation.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	f := filepath.Join(dir, "server.go")
	if err := os.WriteFile(f, []byte("package main\n\nfunc handleAuth() {\n\t// validate token\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "handleAuth", f)
	if err != nil {
		t.Fatalf("nn grep: %v", err)
	}
	if !strings.Contains(out, "Resolve each unread related note") {
		t.Errorf("expected gate language in grep footer; got:\n%s", out)
	}
}

// Assertion: TestGrepCmdRelatedNotesInstruction — nn grep appends nn show instruction after related notes.
func TestGrepCmdRelatedNotesInstruction(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	noteFile := filepath.Join(nbDir, "20260101000000-0002.md")
	if err := os.WriteFile(noteFile, []byte("---\nid: 20260101000000-0002\ntitle: Auth flow design\ntype: concept\nstatus: draft\n---\nhandleAuth is responsible for token validation.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	f := filepath.Join(dir, "server.go")
	if err := os.WriteFile(f, []byte("package main\n\nfunc handleAuth() {\n\t// validate token\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "handleAuth", f)
	if err != nil {
		t.Fatalf("nn grep: %v", err)
	}
	if !strings.Contains(out, "nn show") {
		t.Errorf("expected nn show instruction in related notes output; got:\n%s", out)
	}
	if !strings.Contains(out, "skip-related:") {
		t.Errorf("expected skip-related: in related notes output; got:\n%s", out)
	}
}

// Assertion: TestGrepCmdNoMatch — nn grep with no matches produces no output and exits cleanly.
func TestGrepCmdNoMatch(t *testing.T) {
	_, execute := setupNotebook(t)

	dir := t.TempDir()
	f := filepath.Join(dir, "empty.go")
	if err := os.WriteFile(f, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "nonexistentpattern", f)
	if err != nil {
		t.Fatalf("nn grep no match: %v", err)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected empty output for no match; got:\n%s", out)
	}
}

// Assertion: TestGrepCmdRegex — nn grep treats pattern as a regular expression.
func TestGrepCmdRegex(t *testing.T) {
	_, execute := setupNotebook(t)

	dir := t.TempDir()
	f := filepath.Join(dir, "main.go")
	if err := os.WriteFile(f, []byte("func handleAuth() {}\nfunc handleLogin() {}\nfunc helper() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "handle(Auth|Login)", f, "--context", "0")
	if err != nil {
		t.Fatalf("nn grep regex: %v", err)
	}
	if !strings.Contains(out, "handleAuth") {
		t.Errorf("expected 'handleAuth' in regex output; got:\n%s", out)
	}
	if !strings.Contains(out, "handleLogin") {
		t.Errorf("expected 'handleLogin' in regex output; got:\n%s", out)
	}
	if strings.Contains(out, "helper") {
		t.Errorf("expected 'helper' to be excluded by regex; got:\n%s", out)
	}
}

// Assertion: TestGrepCmdInvalidRegex — nn grep returns an error for invalid regex.
func TestGrepCmdInvalidRegex(t *testing.T) {
	_, execute := setupNotebook(t)

	dir := t.TempDir()
	f := filepath.Join(dir, "main.go")
	if err := os.WriteFile(f, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := execute("grep", "[invalid", f)
	if err == nil {
		t.Error("expected error for invalid regex pattern; got nil")
	}
}

// Assertion: TestGrepCmdMultiplePaths — nn grep accepts multiple path arguments and returns matches from all.
func TestGrepCmdMultiplePaths(t *testing.T) {
	_, execute := setupNotebook(t)

	dir1 := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir1, "a.go"), []byte("func targetFunc() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	dir2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir2, "b.go"), []byte("func targetFunc() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "targetFunc", dir1, dir2)
	if err != nil {
		t.Fatalf("nn grep multi-path: %v", err)
	}
	if !strings.Contains(out, filepath.Join(dir1, "a.go")) {
		t.Errorf("expected match from dir1; got:\n%s", out)
	}
	if !strings.Contains(out, filepath.Join(dir2, "b.go")) {
		t.Errorf("expected match from dir2; got:\n%s", out)
	}
}

// Assertion: TestGrepCmdRespectsGitignore — nn grep skips gitignored files when inside a git repo.
func TestGrepCmdRespectsGitignore(t *testing.T) {
	_, execute := setupNotebook(t)

	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", args, out)
		}
	}
	run("git", "init")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")

	ignored := filepath.Join(dir, "ignored.log")
	if err := os.WriteFile(ignored, []byte("secret pattern\n"), 0644); err != nil {
		t.Fatal(err)
	}
	tracked := filepath.Join(dir, "main.go")
	if err := os.WriteFile(tracked, []byte("// nothing here\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitignore := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(gitignore, []byte("*.log\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", ".")

	out, err := execute("grep", "secret", dir)
	if err != nil {
		t.Fatalf("nn grep: %v", err)
	}
	if strings.Contains(out, "secret") {
		t.Errorf("nn grep should not match gitignored file; got:\n%s", out)
	}
}

// Assertion: TestGrepCmdFallsBackOutsideGitRepo — nn grep collects files via walker when not in a git repo.
func TestGrepCmdFallsBackOutsideGitRepo(t *testing.T) {
	_, execute := setupNotebook(t)

	dir := t.TempDir()
	f := filepath.Join(dir, "main.go")
	if err := os.WriteFile(f, []byte("func findMe() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "findMe", dir)
	if err != nil {
		t.Fatalf("nn grep outside git: %v", err)
	}
	if !strings.Contains(out, "findMe") {
		t.Errorf("expected 'findMe' in output outside git repo; got:\n%s", out)
	}
}

// Assertion: TestCollectFilesSkipsGitDir — collectFiles must not traverse .git directories.
func TestCollectFilesSkipsGitDir(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	gitFile := filepath.Join(gitDir, "COMMIT_EDITMSG")
	if err := os.WriteFile(gitFile, []byte("initial commit\n"), 0644); err != nil {
		t.Fatal(err)
	}
	normalFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(normalFile, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var files []string
	if err := collectFiles(dir, &files); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.Contains(f, ".git") {
			t.Errorf("collectFiles traversed .git directory: found %q", f)
		}
	}
}

// Assertion: TestReadFileLinesSkipsBinary — readFileLines must return nil for binary files.
func TestReadFileLinesSkipsBinary(t *testing.T) {
	dir := t.TempDir()
	binFile := filepath.Join(dir, "data.bin")
	content := []byte("some text\x00more text\n")
	if err := os.WriteFile(binFile, content, 0644); err != nil {
		t.Fatal(err)
	}
	lines, err := readFileLines(binFile)
	if err != nil {
		t.Fatalf("readFileLines returned error: %v", err)
	}
	if lines != nil {
		t.Errorf("readFileLines should return nil for binary file; got %d lines", len(lines))
	}
}

// Assertion: TestGrepCmdTraceableSuggestion — nn grep emits a trace suggestion for matches in gotreesitter-parseable files.
func TestGrepCmdTraceableSuggestion(t *testing.T) {
	_, execute := setupNotebook(t)

	dir := t.TempDir()
	f := filepath.Join(dir, "server.go")
	if err := os.WriteFile(f, []byte("package main\n\nfunc fetchCompanyMappings() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "fetchCompanyMappings", f)
	if err != nil {
		t.Fatalf("nn grep traceable: %v", err)
	}
	if !strings.Contains(out, "nn trace") {
		t.Errorf("expected 'nn trace' suggestion for match in parseable file; got:\n%s", out)
	}
	if !strings.Contains(out, "re-exported symbols") {
		t.Errorf("expected limit disclosure 're-exported symbols' in trace suggestion; got:\n%s", out)
	}
}

// Assertion: TestGrepCmdUnparseableFileNoSuggestion — nn grep does not emit trace suggestion for files with no gotreesitter grammar.
func TestGrepCmdUnparseableFileNoSuggestion(t *testing.T) {
	_, execute := setupNotebook(t)

	dir := t.TempDir()
	f := filepath.Join(dir, "data.bin")
	if err := os.WriteFile(f, []byte("fetchCompanyMappings value\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "fetchCompanyMappings", f)
	if err != nil {
		t.Fatalf("nn grep unparseable: %v", err)
	}
	if strings.Contains(out, "nn trace") {
		t.Errorf("expected no trace suggestion for file with no grammar; got:\n%s", out)
	}
}

// Assertion: TestGrepCmdBM25NoiseThreshold — nn grep suppresses notes with score below minimum threshold.
func TestGrepCmdBM25NoiseThreshold(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// Create a note with very low BM25 overlap (unrelated vocabulary).
	noteFile := filepath.Join(nbDir, "20260101000000-0010.md")
	if err := os.WriteFile(noteFile, []byte("---\nid: 20260101000000-0010\ntitle: Unrelated quantum physics\ntype: concept\nstatus: draft\n---\nquantum entanglement superposition wavefunction collapse.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	f := filepath.Join(dir, "auth.go")
	if err := os.WriteFile(f, []byte("package main\n\nfunc handleAuth() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "handleAuth", f)
	if err != nil {
		t.Fatalf("nn grep noise threshold: %v", err)
	}
	if strings.Contains(out, "Unrelated quantum physics") {
		t.Errorf("expected low-relevance note to be suppressed by threshold; got:\n%s", out)
	}
}

// Assertion: TestGrepCmdContextFlag — nn grep --context N includes surrounding lines in BM25 query.
func TestGrepCmdContextFlag(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	noteFile := filepath.Join(nbDir, "20260101000000-0002.md")
	if err := os.WriteFile(noteFile, []byte("---\nid: 20260101000000-0002\ntitle: Token validation policy\ntype: concept\nstatus: draft\n---\ntoken validation must check expiry.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	f := filepath.Join(dir, "auth.go")
	content := "package main\n\n// validate token expiry\nfunc checkExpiry() bool {\n\treturn true\n}\n"
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "checkExpiry", f, "--context", "2")
	if err != nil {
		t.Fatalf("nn grep --context: %v", err)
	}
	// With context=2, the comment line "validate token expiry" is included, boosting BM25 toward the note.
	if !strings.Contains(out, "Token validation policy") {
		t.Errorf("expected 'Token validation policy' note with --context 2; got:\n%s", out)
	}
}

// Assertion: TestGrepContextLinesDisplayed — nn grep --context N prints N surrounding lines in the output.
func TestGrepContextLinesDisplayed(t *testing.T) {
	_, execute := setupNotebook(t)

	dir := t.TempDir()
	f := filepath.Join(dir, "auth.go")
	content := "package main\n// before line\nfunc checkExpiry() bool {\n// after line\n\treturn true\n}\n"
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "checkExpiry", f, "--context", "1")
	if err != nil {
		t.Fatalf("nn grep --context: %v", err)
	}
	if !strings.Contains(out, "before line") {
		t.Errorf("FAIL: TestGrepContextLinesDisplayed: expected context line 'before line' in output; got:\n%s", out)
	}
	if !strings.Contains(out, "after line") {
		t.Errorf("FAIL: TestGrepContextLinesDisplayed: expected context line 'after line' in output; got:\n%s", out)
	}
}

// Assertion: TestGrepContextSeparator — nn grep --context N prints -- between non-overlapping match groups.
func TestGrepContextSeparator(t *testing.T) {
	_, execute := setupNotebook(t)

	dir := t.TempDir()
	f := filepath.Join(dir, "multi.go")
	// Two matches far apart so context windows (1 line each) don't overlap.
	content := "line1\nMATCH_A\nline3\nline4\nline5\nline6\nline7\nMATCH_B\nline9\n"
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "MATCH_", f, "--context", "1")
	if err != nil {
		t.Fatalf("nn grep --context separator: %v", err)
	}
	// The separator must appear as a line on its own (not inside a URL or suggestion).
	lines := strings.Split(out, "\n")
	found := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "--" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("FAIL: TestGrepContextSeparator: expected '--' on its own line between non-overlapping match groups; got:\n%s", out)
	}
}

// Assertion: TestGrepCmdTraceFlag — nn grep --trace invokes nn trace inline for each traceable matched file.
func TestGrepCmdTraceFlag(t *testing.T) {
	_, execute := setupNotebook(t)

	dir := t.TempDir()
	f := filepath.Join(dir, "server.go")
	if err := os.WriteFile(f, []byte("package main\n\nfunc fetchCompanyMappings() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "--trace", "fetchCompanyMappings", f)
	if err != nil {
		t.Fatalf("nn grep --trace: %v", err)
	}
	if !strings.Contains(out, "fetchCompanyMappings") {
		t.Errorf("expected trace call graph output with symbol name; got:\n%s", out)
	}
	// Call graph output includes a kind marker in parens from trace renderer.
	if !strings.Contains(out, "(function)") && !strings.Contains(out, "(method)") {
		t.Errorf("expected trace kind marker '(function)' or '(method)' in --trace output; got:\n%s", out)
	}
}

// Assertion: TestGrepContextDoesNotCrossFileBoundary — nn grep --context N on a match at line 1 of a non-first file must not panic
func TestGrepContextDoesNotCrossFileBoundary(t *testing.T) {
	_, execute := setupNotebook(t)

	dir := t.TempDir()
	// File A: 10 lines, no match
	fileA := filepath.Join(dir, "a.go")
	var sb strings.Builder
	for i := 0; i < 10; i++ {
		sb.WriteString("// line\n")
	}
	if err := os.WriteFile(fileA, []byte(sb.String()), 0644); err != nil {
		t.Fatal(err)
	}
	// File B: match on line 1
	fileB := filepath.Join(dir, "b.go")
	if err := os.WriteFile(fileB, []byte("package main // TARGET\nfunc foo() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "TARGET", dir, "--context", "4", "--notes-per-match", "0")
	if err != nil {
		t.Fatalf("nn grep panicked or errored: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "TARGET") {
		t.Errorf("expected match 'TARGET' in output; got:\n%s", out)
	}
	// Context must not include lines from file A
	if strings.Contains(out, fileA) {
		t.Errorf("context must not include lines from file A; got:\n%s", out)
	}
}
