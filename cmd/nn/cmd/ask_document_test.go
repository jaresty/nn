package cmd

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/feedback"
)

// property [15a]+[15b]+[16b]+[17a]+[17b]: surface=document dispatches to the
// delegated Plannotator adapter (no hosted server), invokes plannotator with a
// --result-file in the session dir, and records a plannotator-decision envelope.
func TestAskDocumentSurfaceDelegatesToPlannotator(t *testing.T) {
	t.Setenv("NN_CONFIG_DIR", t.TempDir())

	var argv []string
	var sess askSession
	opts := askOptions{
		surface:      "document",
		instructions: "review this",
		out:          io.Discard,
	}
	// runPlannotator is injected; open must NOT be called for the delegated path.
	opts.open = func(string) error {
		t.Fatalf("hosted open hook called for document surface (should delegate)")
		return nil
	}
	opts.runPlannotator = func(argv2 []string) error {
		argv = argv2
		for i, a := range argv2 {
			if a == "--result-file" && i+1 < len(argv2) {
				os.WriteFile(argv2[i+1], []byte(`{"decision":"annotated"}`), 0o644)
			}
		}
		return nil
	}

	sess, err := runAsk(opts)
	if err != nil {
		t.Fatalf("runAsk: %v", err)
	}

	// property [16b]: argv names --result-file at the session path.
	wantResult := filepath.Join(sess.dir, "result.plannotator.json")
	if !containsPair(argv, "--result-file", wantResult) {
		t.Fatalf("argv %v missing --result-file %q", argv, wantResult)
	}

	// property [17a]+[17b]: envelope surface=document, names the plannotator artifact.
	result, err := feedback.ReadResult(sess.dir)
	if err != nil {
		t.Fatalf("ReadResult: %v", err)
	}
	if result.Surface != "document" {
		t.Fatalf("result surface = %q, want document", result.Surface)
	}
	found := false
	for _, a := range result.Artifacts {
		if a.Format == "plannotator-decision" && a.Path == "result.plannotator.json" {
			found = true
		}
	}
	if !found {
		t.Fatalf("result artifacts %+v: no plannotator-decision artifact", result.Artifacts)
	}
}

// property [16a'-ii]: with no --document flag, the adapter writes the
// instructions to <dir>/document.md and annotates that file.
func TestAskDocumentWritesSessionFileWhenNoDocumentFlag(t *testing.T) {
	t.Setenv("NN_CONFIG_DIR", t.TempDir())

	var argv []string
	opts := askOptions{
		surface:      "document",
		instructions: "please review the plan",
		open:         func(string) error { return nil },
		out:          io.Discard,
	}
	var dir string
	opts.runPlannotator = func(a []string) error {
		argv = a
		for i, x := range a {
			if x == "--result-file" && i+1 < len(a) {
				os.WriteFile(a[i+1], []byte(`{}`), 0o644)
			}
		}
		return nil
	}
	sess, err := runAsk(opts)
	if err != nil {
		t.Fatalf("runAsk: %v", err)
	}
	dir = sess.dir
	docPath := filepath.Join(dir, "document.md")
	body, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("document.md not written: %v", err)
	}
	if !strings.Contains(string(body), "please review the plan") {
		t.Fatalf("document.md = %q, want instructions", body)
	}
	if argv[0] != "annotate" || argv[1] != docPath {
		t.Fatalf("argv %v: want annotate %q first", argv, docPath)
	}
}

// property [16a'-i]: when --document is given, that path is annotated directly
// and no session document.md is written.
func TestAskDocumentUsesDocumentFlagWhenGiven(t *testing.T) {
	t.Setenv("NN_CONFIG_DIR", t.TempDir())

	src := filepath.Join(t.TempDir(), "spec.md")
	os.WriteFile(src, []byte("# spec"), 0o644)

	var argv []string
	opts := askOptions{
		surface:  "document",
		document: src,
		open:     func(string) error { return nil },
		out:      io.Discard,
	}
	opts.runPlannotator = func(a []string) error {
		argv = a
		for i, x := range a {
			if x == "--result-file" && i+1 < len(a) {
				os.WriteFile(a[i+1], []byte(`{}`), 0o644)
			}
		}
		return nil
	}
	sess, err := runAsk(opts)
	if err != nil {
		t.Fatalf("runAsk: %v", err)
	}
	if argv[1] != src {
		t.Fatalf("argv %v: want annotate %q", argv, src)
	}
	if _, err := os.Stat(filepath.Join(sess.dir, "document.md")); err == nil {
		t.Fatalf("session document.md written despite --document flag")
	}
}

// property [28]: a URL passed to --document is handed to plannotator's annotate
// argument verbatim (plannotator accepts file, folder, or https:// URL).
func TestAskDocumentPassesURLThrough(t *testing.T) {
	t.Setenv("NN_CONFIG_DIR", t.TempDir())

	const url = "https://example.com/spec.html"
	var argv []string
	opts := askOptions{
		surface:  "document",
		document: url,
		open:     func(string) error { return nil },
		out:      io.Discard,
	}
	opts.runPlannotator = func(a []string) error {
		argv = a
		for i, x := range a {
			if x == "--result-file" && i+1 < len(a) {
				os.WriteFile(a[i+1], []byte(`{}`), 0o644)
			}
		}
		return nil
	}
	sess, err := runAsk(opts)
	if err != nil {
		t.Fatalf("runAsk: %v", err)
	}
	if argv[0] != "annotate" || argv[1] != url {
		t.Fatalf("argv %v: want annotate %q verbatim", argv, url)
	}
	// A URL must not be written as a session file.
	if _, err := os.Stat(filepath.Join(sess.dir, "document.md")); err == nil {
		t.Fatalf("session document.md written for a URL")
	}
}

// property [30a]: local HTML delegates with a session-scoped Plannotator data dir.
// property [30b]: the override is child-local and does not mutate the nn process.
// property [30c]: the ordinary thin result envelope still completes.
func TestAskDocumentLocalHTMLIsolatesPlannotatorHistory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake plannotator executable uses a POSIX shell")
	}
	t.Setenv("NN_CONFIG_DIR", t.TempDir())
	t.Setenv("PLANNOTATOR_DATA_DIR", "/existing/plannotator-data")

	capture := installFakePlannotator(t)
	htmlPath := filepath.Join(t.TempDir(), "review.HTML")
	if err := os.WriteFile(htmlPath, []byte("<!doctype html><h1>review</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}

	sess, err := runAsk(askOptions{surface: "document", document: htmlPath, out: io.Discard})
	if err != nil {
		t.Fatalf("runAsk(local HTML): %v", err)
	}
	got, err := os.ReadFile(capture)
	if err != nil {
		t.Fatalf("read child environment capture: %v", err)
	}
	want := filepath.Join(sess.dir, "plannotator-data")
	if string(got) != want {
		t.Fatalf("PLANNOTATOR_DATA_DIR = %q, want session-scoped %q", got, want)
	}
	if got := os.Getenv("PLANNOTATOR_DATA_DIR"); got != "/existing/plannotator-data" {
		t.Fatalf("parent PLANNOTATOR_DATA_DIR = %q, want unchanged", got)
	}
	result, err := feedback.ReadResult(sess.dir)
	if err != nil {
		t.Fatalf("ReadResult: %v", err)
	}
	if result.Surface != "document" || len(result.Artifacts) != 1 || result.Artifacts[0].Format != "plannotator-decision" {
		t.Fatalf("result envelope = %+v, want unchanged document decision artifact", result)
	}
}

// property [31]: non-HTML documents retain the inherited Plannotator data dir.
func TestAskDocumentMarkdownPreservesPlannotatorHistory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake plannotator executable uses a POSIX shell")
	}
	t.Setenv("NN_CONFIG_DIR", t.TempDir())
	t.Setenv("PLANNOTATOR_DATA_DIR", "/existing/plannotator-data")

	capture := installFakePlannotator(t)
	markdownPath := filepath.Join(t.TempDir(), "review.md")
	if err := os.WriteFile(markdownPath, []byte("# review"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runAsk(askOptions{surface: "document", document: markdownPath, out: io.Discard}); err != nil {
		t.Fatalf("runAsk(markdown): %v", err)
	}
	got, err := os.ReadFile(capture)
	if err != nil {
		t.Fatalf("read child environment capture: %v", err)
	}
	if string(got) != "/existing/plannotator-data" {
		t.Fatalf("markdown PLANNOTATOR_DATA_DIR = %q, want inherited value", got)
	}
}

func installFakePlannotator(t *testing.T) string {
	t.Helper()
	binDir := t.TempDir()
	capture := filepath.Join(t.TempDir(), "plannotator-env.txt")
	script := filepath.Join(binDir, "plannotator")
	body := `#!/bin/sh
printf '%s' "${PLANNOTATOR_DATA_DIR-}" > "$PLANNOTATOR_CAPTURE"
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--result-file" ]; then
    shift
    printf '%s' '{"decision":"annotated","feedback":""}' > "$1"
    exit 0
  fi
  shift
done
exit 1
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PLANNOTATOR_CAPTURE", capture)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return capture
}

// property [29]: the --document flag help names URL as an accepted input.
func TestAskDocumentFlagHelpMentionsURL(t *testing.T) {
	cmd := newAskCmd(&rootState{})
	usage := cmd.Flags().Lookup("document").Usage
	if !strings.Contains(strings.ToLower(usage), "url") {
		t.Fatalf("--document usage %q does not mention URL", usage)
	}
}

func containsPair(argv []string, flag, val string) bool {
	for i, a := range argv {
		if a == flag && i+1 < len(argv) && argv[i+1] == val {
			return true
		}
	}
	return false
}
