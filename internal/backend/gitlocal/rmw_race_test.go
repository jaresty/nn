package gitlocal_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/backend/gitlocal"
	"github.com/jaresty/nn/internal/note"
)

// TestAddLinksConcurrentDropsLinks proves the read-modify-write race in AddLinks:
// two goroutines both read the source note before either writes, so one set of
// links is silently overwritten by the other.
func TestAddLinksConcurrentDropsLinks(t *testing.T) {
	dir := t.TempDir()
	configDir := t.TempDir()
	initGitRepo(t, dir)
	b, err := gitlocal.NewWithConfigDir(dir, configDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Create source and two target notes.
	src := &note.Note{
		ID: note.GenerateID(), Title: "Source", Type: note.TypeConcept, Status: note.StatusDraft,
		Created: time.Now().UTC().Truncate(time.Second), Modified: time.Now().UTC().Truncate(time.Second),
	}
	dst1 := &note.Note{
		ID: note.GenerateID(), Title: "Dest1", Type: note.TypeConcept, Status: note.StatusDraft,
		Created: time.Now().UTC().Truncate(time.Second), Modified: time.Now().UTC().Truncate(time.Second),
	}
	dst2 := &note.Note{
		ID: note.GenerateID(), Title: "Dest2", Type: note.TypeConcept, Status: note.StatusDraft,
		Created: time.Now().UTC().Truncate(time.Second), Modified: time.Now().UTC().Truncate(time.Second),
	}
	for _, n := range []*note.Note{src, dst1, dst2} {
		if err := b.Write(n); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	// Concurrently add one link each from the same source note.
	const attempts = 20
	for range attempts {
		// Reset source to no links before each attempt.
		src.Links = nil
		src.Modified = time.Now().UTC().Truncate(time.Second)
		if err := b.Update(src); err != nil {
			t.Fatalf("reset Update: %v", err)
		}

		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			errs[0] = b.AddLink(src.ID, dst1.ID, "link to 1", "supports", "draft")
		}()
		go func() {
			defer wg.Done()
			errs[1] = b.AddLink(src.ID, dst2.ID, "link to 2", "supports", "draft")
		}()
		wg.Wait()

		// One may error (duplicate detection on re-read), but both links must exist.
		for i, e := range errs {
			if e != nil {
				t.Logf("AddLink %d error (may be duplicate): %v", i, e)
			}
		}

		got, err := b.Read(src.ID)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		linkCount := len(got.Links)
		// After two concurrent AddLink calls to different targets, both links must survive.
		// Without the mutex fix, one will be overwritten and linkCount will be 1.
		if linkCount < 2 {
			t.Errorf("attempt: concurrent AddLink dropped a link — got %d links (want 2): %s",
				linkCount, fmt.Sprintf("%v", got.Links))
			return
		}
	}
}
