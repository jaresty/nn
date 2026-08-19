package cmd

import (
	"io"
	"os"
	"path/filepath"
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

func containsPair(argv []string, flag, val string) bool {
	for i, a := range argv {
		if a == flag && i+1 < len(argv) && argv[i+1] == val {
			return true
		}
	}
	return false
}
