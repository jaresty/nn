package cmd

import (
	"os"
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

	out, err := execute("grep", "handle(Auth|Login)", f)
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
