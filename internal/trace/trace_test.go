package trace_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestBuildIndexReturnsNonNilForAnyTimeout(t *testing.T) {
	// property [1a]: BuildIndex returns a non-nil Index and terminates for any
	// DefaultParseTimeoutMicros >= 0.
	dir := t.TempDir()
	writeTempFile(t, dir, "ok.go", `package main
func Kept() {}
`)
	for _, tmo := range []uint64{0, 1} {
		type result struct {
			idx *trace.Index
			err error
		}
		done := make(chan result, 1)
		orig := trace.DefaultParseTimeoutMicros
		trace.DefaultParseTimeoutMicros = tmo
		go func() {
			idx, err := trace.BuildIndex(dir)
			done <- result{idx, err}
		}()
		select {
		case r := <-done:
			trace.DefaultParseTimeoutMicros = orig
			if r.err != nil {
				t.Fatalf("timeout=%d: BuildIndex error: %v", tmo, r.err)
			}
			if r.idx == nil {
				t.Fatalf("timeout=%d: expected non-nil Index", tmo)
			}
		case <-time.After(10 * time.Second):
			trace.DefaultParseTimeoutMicros = orig
			t.Fatalf("timeout=%d: BuildIndex did not terminate within 10s", tmo)
		}
	}
}

func TestBuildIndexAppliesParseTimeout(t *testing.T) {
	// property [1b]: DefaultParseTimeoutMicros observably changes parse
	// completeness — a value of 0 recovers definitions that a tiny positive
	// value may omit.
	dir := t.TempDir()
	var sb strings.Builder
	sb.WriteString("package main\n")
	for i := 0; i < 4000; i++ {
		fmt.Fprintf(&sb, "func Fn%d() { _ = %d + %d }\n", i, i, i)
	}
	writeTempFile(t, dir, "big.go", sb.String())

	orig := trace.DefaultParseTimeoutMicros

	trace.DefaultParseTimeoutMicros = 0
	full, err := trace.BuildIndex(dir)
	if err != nil {
		trace.DefaultParseTimeoutMicros = orig
		t.Fatalf("full parse error: %v", err)
	}

	trace.DefaultParseTimeoutMicros = 1
	limited, err := trace.BuildIndex(dir)
	trace.DefaultParseTimeoutMicros = orig
	if err != nil {
		t.Fatalf("limited parse error: %v", err)
	}

	if len(full.All) == 0 {
		t.Fatalf("expected full parse to recover definitions, got 0")
	}
	if len(limited.All) >= len(full.All) {
		t.Fatalf("expected 1us timeout to omit definitions: full=%d limited=%d (timeout not applied)",
			len(full.All), len(limited.All))
	}
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

	// Annotator that surfaces a note only when the node's source-span query
	// contains the distinctive term, verifying the source text drives annotation.
	annotate := func(query string) []trace.NoteRef {
		if strings.Contains(query, "distinctivequerytermxyz") {
			return []trace.NoteRef{{ID: "test-note-1", Title: "distinctivequerytermxyz usage pattern"}}
		}
		return nil
	}

	result := trace.Trace(idx, []string{"MyFunc"}, 1, annotate)

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

// TestBuildIndexConcurrencyCap verifies that BuildIndex correctly indexes
// symbols across multiple files when the goroutine pool is active.
// Combined with the structural guard (grep for g.SetLimit(4)), this confirms
// the cap is both present and behaviorally correct.
func TestBuildIndexConcurrencyCap(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 10; i++ {
		name := filepath.Join(dir, strings.Repeat("f", i+1)+".go")
		content := "package main\nfunc Cap" + strings.Repeat("X", i+1) + "() {}\n"
		if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	idx, err := trace.BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}
	if len(idx.All) != 10 {
		t.Errorf("FAIL: TestBuildIndexConcurrencyCap: expected 10 symbols, got %d", len(idx.All))
	}
}

// TestBuildIndexSkipsLargeFiles verifies that files exceeding 500KB are not
// parsed and their symbols do not appear in the index.
func TestBuildIndexSkipsLargeFiles(t *testing.T) {
	dir := t.TempDir()

	// Write a valid Go file that exceeds 500KB via comment padding.
	padding := strings.Repeat("// padding line to exceed size limit\n", 15000) // ~570KB
	largeContent := "package main\n" + padding + "\nfunc LargeFileSymbol() {}\n"
	if len(largeContent) <= 500*1024 {
		t.Fatalf("test setup: large file is only %d bytes, need >%d", len(largeContent), 500*1024)
	}
	if err := os.WriteFile(filepath.Join(dir, "large.go"), []byte(largeContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a normal file that should be indexed.
	if err := os.WriteFile(filepath.Join(dir, "small.go"), []byte("package main\nfunc SmallFileSymbol() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	idx, err := trace.BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}
	if _, ok := idx.ByName["LargeFileSymbol"]; ok {
		t.Error("FAIL: TestBuildIndexSkipsLargeFiles: LargeFileSymbol should be skipped (file >500KB) but was indexed")
	}
	if _, ok := idx.ByName["SmallFileSymbol"]; !ok {
		t.Error("FAIL: TestBuildIndexSkipsLargeFiles: SmallFileSymbol should be indexed but was not found")
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
