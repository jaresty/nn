package gitlocal_test

import (
	"sync"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// TestRemoveLinkByTypeConcurrentRace proves the RMW race in RemoveLinkByType.
func TestRemoveLinkByTypeConcurrentRace(t *testing.T) {
	b := setupBackend(t)
	src, dst1, dst2 := makeNote("src"), makeNote("dst1"), makeNote("dst2")
	for _, n := range []*note.Note{src, dst1, dst2} {
		if err := b.Write(n); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	const attempts = 20
	for range attempts {
		src.Links = []note.Link{
			{TargetID: dst1.ID, Annotation: "a", Type: "supports", Status: "draft"},
			{TargetID: dst2.ID, Annotation: "b", Type: "refines", Status: "draft"},
		}
		src.Modified = time.Now().UTC().Truncate(time.Second)
		if err := b.Update(src); err != nil {
			t.Fatalf("setup: %v", err)
		}
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); _ = b.RemoveLinkByType(src.ID, dst1.ID, "supports") }()
		go func() { defer wg.Done(); _ = b.RemoveLinkByType(src.ID, dst2.ID, "refines") }()
		wg.Wait()
		got, err := b.Read(src.ID)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if len(got.Links) > 0 {
			t.Errorf("RemoveLinkByType race: got %d links (want 0)", len(got.Links))
			return
		}
	}
}
