package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// grepTraceFixture writes a Go file where a caller invokes a receiver method
// whose bare name is defined on two different receivers, so name-only
// resolution produces an ambiguous edge with two candidates. It also contains a
// call to an undefined function to exercise the unresolved path.
func grepTraceFixture(t *testing.T) (func(...string) (string, error), string) {
	t.Helper()
	_, execute := setupNotebook(t)
	dir := t.TempDir()
	src := `package server

type Backend struct{}
type Cache struct{}

func (Backend) Save() {}
func (Cache) Save() {}

func handleAuth(b Backend) {
	b.Save()
	missingHelper()
}
`
	f := filepath.Join(dir, "server.go")
	if err := os.WriteFile(f, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return execute, f
}

// property [1]: an ambiguous-receiver node's grep --trace line surfaces the
// receiver and that multiple candidates share the name.
func TestGrepTraceLabelsAmbiguousReceiver(t *testing.T) {
	execute, f := grepTraceFixture(t)
	out, err := execute("grep", "handleAuth", f, "--trace")
	if err != nil {
		t.Fatalf("nn grep --trace: %v", err)
	}
	// Both Save candidates appear (keep all), and the ambiguity is labeled with
	// the receiver and a candidate count.
	if !strings.Contains(out, "Save") {
		t.Fatalf("expected Save candidate in trace output; got:\n%s", out)
	}
	if !strings.Contains(out, "candidates") {
		t.Fatalf("expected ambiguous candidate marker in trace output; got:\n%s", out)
	}
	if !strings.Contains(out, "Backend") {
		t.Fatalf("expected receiver 'Backend' surfaced for ambiguous call; got:\n%s", out)
	}
}

// property [3]: an unresolved call (no matching definition) is surfaced in
// grep --trace output, marked unresolved.
func TestGrepTraceShowsUnresolved(t *testing.T) {
	execute, f := grepTraceFixture(t)
	out, err := execute("grep", "handleAuth", f, "--trace")
	if err != nil {
		t.Fatalf("nn grep --trace: %v", err)
	}
	if !strings.Contains(out, "missingHelper") {
		t.Fatalf("expected unresolved call 'missingHelper' in trace output; got:\n%s", out)
	}
	if !strings.Contains(out, "unresolved") {
		t.Fatalf("expected 'unresolved' marker in trace output; got:\n%s", out)
	}
}
