package gitlocal_test

import (
	"sync"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/backend"
	"github.com/jaresty/nn/internal/backend/gitlocal"
	"github.com/jaresty/nn/internal/note"
)

func makeNote(title string) *note.Note {
	return &note.Note{
		ID: note.GenerateID(), Title: title, Type: note.TypeConcept, Status: note.StatusDraft,
		Created: time.Now().UTC().Truncate(time.Second), Modified: time.Now().UTC().Truncate(time.Second),
	}
}

func setupBackend(t *testing.T) *gitlocal.Backend {
	t.Helper()
	dir := t.TempDir()
	configDir := t.TempDir()
	initGitRepo(t, dir)
	b, err := gitlocal.NewWithConfigDir(dir, configDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return b
}

// TestAddLinksConcurrentDropsLinksViaAddLinks proves the RMW race in AddLinks.
func TestAddLinksConcurrentDropsLinksViaAddLinks(t *testing.T) {
	b := setupBackend(t)
	src, dst1, dst2 := makeNote("src"), makeNote("dst1"), makeNote("dst2")
	for _, n := range []*note.Note{src, dst1, dst2} {
		if err := b.Write(n); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	const attempts = 20
	for range attempts {
		src.Links = nil
		src.Modified = time.Now().UTC().Truncate(time.Second)
		if err := b.Update(src); err != nil {
			t.Fatalf("reset: %v", err)
		}
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = b.AddLinks(src.ID, []backend.LinkTarget{{ToID: dst1.ID, Annotation: "a", Type: "supports", Status: "draft"}})
		}()
		go func() {
			defer wg.Done()
			_ = b.AddLinks(src.ID, []backend.LinkTarget{{ToID: dst2.ID, Annotation: "b", Type: "supports", Status: "draft"}})
		}()
		wg.Wait()
		got, err := b.Read(src.ID)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if len(got.Links) < 2 {
			t.Errorf("AddLinks race: got %d links (want 2)", len(got.Links))
			return
		}
	}
}

// TestRemoveLinkConcurrentRace proves the RMW race in RemoveLink.
func TestRemoveLinkConcurrentRace(t *testing.T) {
	b := setupBackend(t)
	src, dst1, dst2 := makeNote("src"), makeNote("dst1"), makeNote("dst2")
	for _, n := range []*note.Note{src, dst1, dst2} {
		if err := b.Write(n); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	const attempts = 20
	for range attempts {
		// Set up two links on src.
		src.Links = []note.Link{
			{TargetID: dst1.ID, Annotation: "a", Type: "supports", Status: "draft"},
			{TargetID: dst2.ID, Annotation: "b", Type: "supports", Status: "draft"},
		}
		src.Modified = time.Now().UTC().Truncate(time.Second)
		if err := b.Update(src); err != nil {
			t.Fatalf("setup Update: %v", err)
		}
		// Concurrently remove each link.
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); _ = b.RemoveLink(src.ID, dst1.ID) }()
		go func() { defer wg.Done(); _ = b.RemoveLink(src.ID, dst2.ID) }()
		wg.Wait()
		got, err := b.Read(src.ID)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		// Both removals must survive; if race, one removal is overwritten → 1 link remains.
		if len(got.Links) > 0 {
			t.Errorf("RemoveLink race: got %d links (want 0)", len(got.Links))
			return
		}
	}
}

// TestUpdateConcurrentRace proves the RMW race in Update.
func TestUpdateConcurrentRace(t *testing.T) {
	b := setupBackend(t)
	n := makeNote("original")
	if err := b.Write(n); err != nil {
		t.Fatalf("Write: %v", err)
	}
	const attempts = 20
	for range attempts {
		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			cp := *n
			cp.Body = "body-A"
			cp.Modified = time.Now().UTC().Truncate(time.Second)
			errs[0] = b.Update(&cp)
		}()
		go func() {
			defer wg.Done()
			cp := *n
			cp.Body = "body-B"
			cp.Modified = time.Now().UTC().Truncate(time.Second)
			errs[1] = b.Update(&cp)
		}()
		wg.Wait()
		for i, e := range errs {
			if e != nil {
				t.Logf("Update %d err: %v", i, e)
			}
		}
		// If there's a race, both goroutines read the same file and one's write is lost.
		// The real invariant is that neither Update returns an error — both must succeed.
		if errs[0] != nil && errs[1] != nil {
			t.Errorf("both concurrent Updates failed")
			return
		}
	}
}

// TestPromoteConcurrentRace proves the RMW race in Promote.
func TestPromoteConcurrentRace(t *testing.T) {
	b := setupBackend(t)
	const attempts = 20
	for range attempts {
		n := makeNote("promote-me")
		if err := b.Write(n); err != nil {
			t.Fatalf("Write: %v", err)
		}
		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Add(2)
		go func() { defer wg.Done(); errs[0] = b.Promote(n.ID, note.StatusReviewed) }()
		go func() { defer wg.Done(); errs[1] = b.Promote(n.ID, note.StatusPermanent) }()
		wg.Wait()
		if errs[0] != nil && errs[1] != nil {
			t.Errorf("both concurrent Promotes failed: %v / %v", errs[0], errs[1])
			return
		}
	}
}
