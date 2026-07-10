package gitlocal_test

import (
	"sync"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/backend"
	"github.com/jaresty/nn/internal/note"
)

// TestBulkUpdateLinksConcurrentRace proves the RMW race in BulkUpdateLinks.
func TestBulkUpdateLinksConcurrentRace(t *testing.T) {
	b := setupBackend(t)
	src, dst1, dst2 := makeNote("src"), makeNote("dst1"), makeNote("dst2")
	for _, n := range []*note.Note{src, dst1, dst2} {
		if err := b.Write(n); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	if err := b.AddLink(src.ID, dst1.ID, "orig1", "supports", "draft"); err != nil {
		t.Fatalf("AddLink dst1: %v", err)
	}
	if err := b.AddLink(src.ID, dst2.ID, "orig2", "supports", "draft"); err != nil {
		t.Fatalf("AddLink dst2: %v", err)
	}

	const attempts = 20
	ann1, ann2 := "updated-ann1", "updated-ann2"
	for range attempts {
		// Reset annotations.
		got, _ := b.Read(src.ID)
		for i := range got.Links {
			got.Links[i].Annotation = "orig"
		}
		got.Modified = time.Now().UTC().Truncate(time.Second)
		if err := b.Update(got); err != nil {
			t.Fatalf("reset: %v", err)
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = b.BulkUpdateLinks(src.ID, []backend.LinkUpdate{{ToID: dst1.ID, Annotation: &ann1}})
		}()
		go func() {
			defer wg.Done()
			_ = b.BulkUpdateLinks(src.ID, []backend.LinkUpdate{{ToID: dst2.ID, Annotation: &ann2}})
		}()
		wg.Wait()

		got, err := b.Read(src.ID)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		// Both updates must survive; if race, one is overwritten back to "orig".
		updated := 0
		for _, lnk := range got.Links {
			if lnk.Annotation == ann1 || lnk.Annotation == ann2 {
				updated++
			}
		}
		if updated < 2 {
			t.Errorf("BulkUpdateLinks race: only %d of 2 annotations updated", updated)
			return
		}
	}
}

// TestUpdateLinkConcurrentRace proves the RMW race in UpdateLink.
func TestUpdateLinkConcurrentRace(t *testing.T) {
	b := setupBackend(t)
	src, dst1, dst2 := makeNote("src"), makeNote("dst1"), makeNote("dst2")
	for _, n := range []*note.Note{src, dst1, dst2} {
		if err := b.Write(n); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	if err := b.AddLink(src.ID, dst1.ID, "orig1", "supports", "draft"); err != nil {
		t.Fatalf("AddLink dst1: %v", err)
	}
	if err := b.AddLink(src.ID, dst2.ID, "orig2", "supports", "draft"); err != nil {
		t.Fatalf("AddLink dst2: %v", err)
	}

	const attempts = 20
	ann1, ann2 := "upd1", "upd2"
	for range attempts {
		got, _ := b.Read(src.ID)
		for i := range got.Links {
			got.Links[i].Annotation = "orig"
		}
		got.Modified = time.Now().UTC().Truncate(time.Second)
		if err := b.Update(got); err != nil {
			t.Fatalf("reset: %v", err)
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = b.UpdateLink(src.ID, dst1.ID, &ann1, nil, nil)
		}()
		go func() {
			defer wg.Done()
			_ = b.UpdateLink(src.ID, dst2.ID, &ann2, nil, nil)
		}()
		wg.Wait()

		got, err := b.Read(src.ID)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		updated := 0
		for _, lnk := range got.Links {
			if lnk.Annotation == ann1 || lnk.Annotation == ann2 {
				updated++
			}
		}
		if updated < 2 {
			t.Errorf("UpdateLink race: only %d of 2 annotations updated", updated)
			return
		}
	}
}
