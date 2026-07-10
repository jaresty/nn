package gitlocal_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/backend/gitlocal"
	"github.com/jaresty/nn/internal/note"
)

// TestParallelUpdates verifies that concurrent Write calls on the same backend
// all succeed — none fail due to git index.lock contention.
func TestParallelUpdates(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	b, err := gitlocal.New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	const n = 5
	notes := make([]*note.Note, n)
	for i := range n {
		nn := &note.Note{
			ID:       note.GenerateID(),
			Title:    fmt.Sprintf("Parallel Note %d", i),
			Type:     note.TypeConcept,
			Status:   note.StatusDraft,
			Created:  time.Now().UTC().Truncate(time.Second),
			Modified: time.Now().UTC().Truncate(time.Second),
			Body:     "initial",
		}
		if err := b.Write(nn); err != nil {
			t.Fatalf("setup Write %d: %v", i, err)
		}
		notes[i] = nn
	}

	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			nn := notes[idx]
			nn.Body = fmt.Sprintf("updated body %d", idx)
			errs[idx] = b.Update(nn)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("parallel Update %d failed: %v", i, err)
		}
	}
}
