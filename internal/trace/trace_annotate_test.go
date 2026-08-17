package trace_test

import (
	"testing"

	"github.com/jaresty/nn/internal/trace"
)

// property [2]: each node's NNNotes are exactly what the supplied annotate
// function returns for that node's source-span query.
// property [3]: a nil annotate yields empty NNNotes on every node.
func TestTraceAnnotateDelegation(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "simple.go", `package main
func Alpha() { Beta() }
func Beta() {}
`)
	idx, err := trace.BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	// property [2]: annotate is called with the node source query; its return
	// value is attached verbatim as the node's NNNotes. We stamp a sentinel note
	// so we can assert the delegation wired through.
	var seenQueries []string
	annotate := func(query string) []trace.NoteRef {
		seenQueries = append(seenQueries, query)
		return []trace.NoteRef{{ID: "stamped", Title: "stamped-note"}}
	}
	result := trace.Trace(idx, []string{"Alpha"}, 3, annotate)
	if len(result.Nodes) == 0 {
		t.Fatal("expected trace nodes")
	}
	for _, n := range result.Nodes {
		if !n.Resolved {
			continue
		}
		if len(n.NNNotes) != 1 || n.NNNotes[0].ID != "stamped" {
			t.Fatalf("node %s: NNNotes not sourced from annotate; got %+v", n.Name, n.NNNotes)
		}
	}
	if len(seenQueries) == 0 {
		t.Fatal("annotate was never called; NNNotes not delegated")
	}
	// The Alpha node's query must contain its own source text.
	foundAlphaSrc := false
	for _, q := range seenQueries {
		if containsAll(q, "Alpha", "Beta") {
			foundAlphaSrc = true
		}
	}
	if !foundAlphaSrc {
		t.Fatalf("annotate not called with node source span; queries=%v", seenQueries)
	}

	// property [3]: nil annotate => empty NNNotes.
	resultNil := trace.Trace(idx, []string{"Alpha"}, 3, nil)
	for _, n := range resultNil.Nodes {
		if len(n.NNNotes) != 0 {
			t.Fatalf("nil annotate: node %s has non-empty NNNotes %+v", n.Name, n.NNNotes)
		}
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		found := false
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
