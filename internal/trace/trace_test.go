package trace_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
