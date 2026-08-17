package trace_test

import (
	"testing"

	"github.com/jaresty/nn/internal/trace"
)

// property [1]: unresolved edges for Go predeclared builtins are not emitted.
// property [2]: real unresolved calls (non-builtins) are still surfaced.
func TestTraceFiltersGoBuiltins(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "b.go", `package main
func Entry() {
	xs := make([]int, 0)
	xs = append(xs, len(xs))
	SomethingUndefined()
}
`)
	idx, err := trace.BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}
	result := trace.Trace(idx, []string{"Entry"}, 3, nil)

	unresolved := map[string]bool{}
	for _, e := range result.Edges {
		if !e.Resolved {
			unresolved[e.To] = true
		}
	}
	for _, b := range []string{"make", "append", "len"} {
		if unresolved[b] {
			t.Fatalf("Go builtin %q emitted as an unresolved edge; want filtered", b)
		}
	}
	if !unresolved["SomethingUndefined"] {
		t.Fatalf("real unresolved call SomethingUndefined was dropped; unresolved=%v", unresolved)
	}
}
