package trace_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
	"github.com/jaresty/nn/internal/trace"
)

// writeTempFile writes content to a temp file with the given name suffix and returns its path.
func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBuildIndex(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "foo.go", `package main
func HelloWorld() {}
func GoodbyeWorld() {}
`)
	idx, err := trace.BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}
	if len(idx.ByName) == 0 {
		t.Error("expected byName entries, got empty index")
	}
	if _, ok := idx.ByName["HelloWorld"]; !ok {
		t.Error("expected HelloWorld in index")
	}
}

func TestDFSCycleGuard(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "cycle.go", `package main
func A() { B() }
func B() { A() }
`)
	idx, err := trace.BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}
	result := trace.Trace(idx, []string{"A"}, 5, nil)
	// Must not infinite-loop; must contain "already expanded" for cycle
	found := false
	for _, n := range result.Nodes {
		if strings.Contains(n.CycleMarker, "already expanded") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'already expanded' cycle guard marker in result nodes")
	}
}

func TestTraceAmbiguousReceiver(t *testing.T) {
	dir := t.TempDir()
	// Two definitions of "Run" — one free function, one method.
	// Caller calls obj.Run() — receiver="obj", N=2 candidates.
	writeTempFile(t, dir, "ambig.go", `package main
func Run() {}
type Worker struct{}
func (w *Worker) Run() {}
func Start() {
	var obj Worker
	obj.Run()
}
`)
	idx, err := trace.BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}
	result := trace.Trace(idx, []string{"Start"}, 3, nil)

	var ambigNode *trace.Node
	for i := range result.Nodes {
		if result.Nodes[i].AmbiguousReceiver {
			ambigNode = &result.Nodes[i]
			break
		}
	}
	if ambigNode == nil {
		t.Fatal("expected at least one node with AmbiguousReceiver=true")
	}
	if ambigNode.Receiver == "" {
		t.Error("expected non-empty Receiver on ambiguous node")
	}
}

func TestTraceAmbiguousReceiverString(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "ambig2.go", `package main
func Run() {}
type Worker struct{}
func (w *Worker) Run() {}
func Start() {
	var obj Worker
	obj.Run()
}
`)
	idx, err := trace.BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}
	result := trace.Trace(idx, []string{"Start"}, 3, nil)
	for _, n := range result.Nodes {
		if n.AmbiguousReceiver && n.Receiver == "" {
			t.Errorf("ambiguous node %s has empty Receiver string", n.Name)
		}
	}
}

func TestTraceNonAmbiguousUnaffected(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "clean.go", `package main
func BuildIndex() {}
func Main() { BuildIndex() }
`)
	idx, err := trace.BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}
	result := trace.Trace(idx, []string{"Main"}, 3, nil)
	for _, n := range result.Nodes {
		if n.AmbiguousReceiver {
			t.Errorf("unexpected AmbiguousReceiver on node %s", n.Name)
		}
	}
}

// TestBuildIndexParallelCorrectness verifies that a parallelized BuildIndex
// produces the same symbol count as the serial implementation.
// The -race flag will catch any mutex gaps introduced during parallelization.
func TestBuildIndexParallelCorrectness(t *testing.T) {
	dir := t.TempDir()
	// Create enough files to exercise the goroutine pool across multiple workers.
	for i := 0; i < 20; i++ {
		name := filepath.Join(dir, filepath.Base(t.TempDir())+".go")
		content := "package main\nfunc Sym" + strings.Repeat("X", i+1) + "() {}\n"
		if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	idx, err := trace.BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}
	// Each file defines exactly one function; expect 20 symbols.
	if len(idx.All) != 20 {
		t.Errorf("FAIL: TestBuildIndexParallelCorrectness: expected 20 symbols, got %d", len(idx.All))
	}
}

// TestTraceBM25UsesSourceText verifies that BM25 note matching in Trace uses the
// full symbol source body, not just the symbol name. It does this by creating a
// note whose title matches a term that appears only in the function body (not the
// name), and asserting that term surfaces as a matched note on the traced node.
func TestTraceBM25UsesSourceText(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "example.go", `package main
func MyFunc() {
	// distinctivequerytermxyz
	_ = 42
}
`)
	idx, err := trace.BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	// Synthetic note whose title contains the distinctive body term.
	notes := []*note.Note{
		{
			ID:    "test-note-1",
			Title: "distinctivequerytermxyz usage pattern",
			Body:  "This note is about distinctivequerytermxyz.",
		},
	}

	result := trace.Trace(idx, []string{"MyFunc"}, 1, notes)

	var myFuncNode *trace.Node
	for i := range result.Nodes {
		if result.Nodes[i].Name == "MyFunc" {
			myFuncNode = &result.Nodes[i]
			break
		}
	}
	if myFuncNode == nil {
		t.Fatal("MyFunc node not found in trace result")
	}
	if len(myFuncNode.NNNotes) == 0 {
		t.Errorf("FAIL: TestTraceBM25UsesSourceText: expected BM25 to surface note via body term 'distinctivequerytermxyz', got no notes on MyFunc node")
	}
}

func BenchmarkBuildIndex(b *testing.B) {
	// Benchmark against the actual nn source tree — representative real-world workload.
	dir := "../.."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := trace.BuildIndex(dir); err != nil {
			b.Fatal(err)
		}
	}
}

func TestJSONOutput(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "simple.go", `package main
func Alpha() { Beta() }
func Beta() {}
`)
	idx, err := trace.BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}
	result := trace.Trace(idx, []string{"Alpha"}, 3, nil)
	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}
	if _, ok := out["nodes"]; !ok {
		t.Error("expected nn_notes key in JSON output — missing nodes")
	}
}
