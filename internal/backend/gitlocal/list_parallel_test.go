package gitlocal

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestReadFilesConcurrentlyOrdering verifies that readFilesConcurrently returns
// file contents in the same order as the input paths slice.
func TestReadFilesConcurrentlyOrdering(t *testing.T) {
	dir := t.TempDir()
	paths := make([]string, 10)
	for i := range paths {
		name := filepath.Join(dir, filepath.FromSlash("file"+string(rune('a'+i))+".txt"))
		if err := os.WriteFile(name, []byte(name), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		paths[i] = name
	}

	results, err := readFilesConcurrently(paths)
	if err != nil {
		t.Fatalf("readFilesConcurrently: %v", err)
	}
	if len(results) != len(paths) {
		t.Fatalf("got %d results, want %d", len(results), len(paths))
	}
	for i, got := range results {
		if string(got) != paths[i] {
			t.Errorf("result[%d] = %q, want %q", i, got, paths[i])
		}
	}
}

// TestReadFilesConcurrentlyErrorPropagation verifies that a missing file causes an error.
func TestReadFilesConcurrentlyErrorPropagation(t *testing.T) {
	_, err := readFilesConcurrently([]string{"/nonexistent/path/file.txt"})
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// TestReadFilesConcurrentlyEmpty verifies empty input returns empty output.
func TestReadFilesConcurrentlyEmpty(t *testing.T) {
	results, err := readFilesConcurrently(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

// TestListOrderingMatchesReadDir verifies List() returns notes in os.ReadDir order.
func TestListOrderingMatchesReadDir(t *testing.T) {
	b, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	entries, _ := os.ReadDir(b.dir)
	var mdFiles []string
	for _, e := range entries {
		if !e.IsDir() {
			mdFiles = append(mdFiles, e.Name())
		}
	}
	if !sort.StringsAreSorted(mdFiles) {
		t.Error("ReadDir is not sorted — ordering invariant broken")
	}
}
