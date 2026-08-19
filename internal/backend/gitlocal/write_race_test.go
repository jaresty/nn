package gitlocal_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/jaresty/nn/internal/backend/gitlocal"
	"github.com/jaresty/nn/internal/note"
)

// property [32b]: concurrent creates never lose a note to a same-path overwrite.
// Even when many writes collide on IDs, every distinct note ends up on disk
// under some ID (the exclusive create forces losers to retry with a fresh ID).
func TestConcurrentWritesNeverOverwrite(t *testing.T) {
	dir := t.TempDir()
	configDir := t.TempDir()
	initGitRepo(t, dir)
	b, err := gitlocal.NewWithConfigDir(dir, configDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	const goroutines = 50
	var wg sync.WaitGroup
	titles := make([]string, goroutines)
	for i := 0; i < goroutines; i++ {
		titles[i] = fmt.Sprintf("concurrent-title-%d", i)
	}
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(title string) {
			defer wg.Done()
			n := &note.Note{ID: note.GenerateID(), Title: title, Type: note.TypeObservation, Status: "draft"}
			if err := b.Write(n); err != nil {
				t.Errorf("Write %q: %v", title, err)
			}
		}(titles[i])
	}
	wg.Wait()

	notes, err := b.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	seen := map[string]bool{}
	for _, n := range notes {
		seen[n.Title] = true
	}
	for _, want := range titles {
		if !seen[want] {
			t.Fatalf("note %q lost — %d/%d notes survived", want, len(seen), goroutines)
		}
	}
}
