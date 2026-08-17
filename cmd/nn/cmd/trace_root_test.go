package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// property [1]: with --root set, nn trace indexes the whole root, so a call to a
// function defined in a sibling directory resolves (appears as a resolved child,
// not an unresolved leaf).
// property [2]: without --root, only the target dir is indexed, so the
// sibling-package call stays unresolved (current behavior preserved).
func TestTraceRootFlagResolvesSiblingPackage(t *testing.T) {
	_, execute := setupNotebook(t)

	proj := t.TempDir()
	pkgA := filepath.Join(proj, "a")
	pkgB := filepath.Join(proj, "b")
	if err := os.MkdirAll(pkgA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pkgB, 0o755); err != nil {
		t.Fatal(err)
	}
	// a.Caller calls SiblingTarget, which is defined only in package b.
	if err := os.WriteFile(filepath.Join(pkgA, "a.go"),
		[]byte("package a\n\nfunc Caller() { SiblingTarget() }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgB, "b.go"),
		[]byte("package b\n\nfunc SiblingTarget() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// property [1]: with --root proj, SiblingTarget resolves — it appears as a
	// traced node line (indented tree node), not just an unresolved leaf.
	withRoot, err := execute("trace", pkgA, "--symbol", "Caller", "--root", proj)
	if err != nil {
		t.Fatalf("nn trace --root: %v", err)
	}
	if !strings.Contains(withRoot, "SiblingTarget (function)") {
		t.Fatalf("--root: expected SiblingTarget resolved as a function node; got:\n%s", withRoot)
	}

	// property [2]: without --root, indexing only pkgA leaves SiblingTarget
	// unresolved (never resolved to a definition node).
	noRoot, err := execute("trace", pkgA, "--symbol", "Caller", "--show-unresolved")
	if err != nil {
		t.Fatalf("nn trace: %v", err)
	}
	if strings.Contains(noRoot, "SiblingTarget (function)") {
		t.Fatalf("without --root: SiblingTarget should be unresolved, but resolved; got:\n%s", noRoot)
	}
}
