package trace_test

import (
	"sort"
	"testing"

	"github.com/jaresty/nn/internal/trace"
)

// property [1]: caching per-file parse results must not change the call graph.
// Multiple functions in one file cause extractCallsInDef to run repeatedly on
// the same source; this pins the resolved edge set so a parse cache that
// mis-attributes calls to the wrong def is caught.
func TestTraceParseCacheEdgesUnchanged(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "multi.go", `package main
func A() { B(); C() }
func B() { C() }
func C() {}
func D() { A() }
`)
	idx, err := trace.BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}
	result := trace.Trace(idx, []string{"D"}, 5, nil)

	// Each function's calls must be attributed to that function, not bled across
	// defs by a bad cache. Collect resolved edges by caller name.
	got := map[string][]string{}
	byID := map[string]string{} // node ID -> name
	for _, n := range result.Nodes {
		byID[n.ID] = n.Name
	}
	for _, e := range result.Edges {
		if !e.Resolved {
			continue
		}
		got[byID[e.From]] = append(got[byID[e.From]], byID[e.To])
	}
	for k := range got {
		sort.Strings(got[k])
	}

	want := map[string][]string{
		"D": {"A"},
		"A": {"B", "C"},
		"B": {"C"},
	}
	for caller, wantCallees := range want {
		g := got[caller]
		if len(g) != len(wantCallees) {
			t.Fatalf("caller %s: edges %v, want %v", caller, g, wantCallees)
		}
		for i := range wantCallees {
			if g[i] != wantCallees[i] {
				t.Fatalf("caller %s: edges %v, want %v", caller, g, wantCallees)
			}
		}
	}
}
