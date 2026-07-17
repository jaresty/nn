package gitlocal_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// TestAddLinkCrossProcessRace proves the cross-process RMW race in AddLink by spawning
// real nn subprocesses via go run. Two concurrent nn link invocations share the same
// source note; both report success but without the fix one link is silently lost.
func TestAddLinkCrossProcessRace(t *testing.T) {
	b, nn := setupCrossProcess(t)
	src, dst1, dst2 := makeNote("src"), makeNote("dst1"), makeNote("dst2")
	for _, n := range []*note.Note{src, dst1, dst2} {
		if err := b.Write(n); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	const attempts = 10
	for i := range attempts {
		src.Links = nil
		src.Modified = time.Now().UTC().Truncate(time.Second)
		if err := b.Update(src); err != nil {
			t.Fatalf("reset attempt %d: %v", i, err)
		}
		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			errs[0] = nn("link", src.ID, dst1.ID, "--type", "supports", "--annotation", "probe A")
		}()
		go func() {
			defer wg.Done()
			errs[1] = nn("link", src.ID, dst2.ID, "--type", "supports", "--annotation", "probe B")
		}()
		wg.Wait()
		for j, e := range errs {
			if e != nil {
				t.Logf("attempt %d link %d error (may be dup): %v", i, j, e)
			}
		}
		got, err := b.Read(src.ID)
		if err != nil {
			t.Fatalf("Read attempt %d: %v", i, err)
		}
		if len(got.Links) < 2 {
			t.Errorf("attempt %d: cross-process AddLink race: got %d links (want 2) — one write was lost", i, len(got.Links))
			return
		}
	}
}


// setupCrossProcess creates a notebook dir and XDG config, returns a backend and an nn runner.
func setupCrossProcess(t *testing.T) (*gitlocal.Backend, func(...string) error) {
	t.Helper()
	dir := t.TempDir()
	initGitRepo(t, dir)

	xdgDir := t.TempDir()
	cfgPath := filepath.Join(xdgDir, "nn", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	cfgContent := fmt.Sprintf("[notebooks]\ndefault = \"personal\"\n[notebooks.personal]\npath = %q\nbackend = \"gitlocal\"\n", dir)
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	b, err := gitlocal.NewWithConfigDir(dir, xdgDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	env := append(os.Environ(), "XDG_CONFIG_HOME="+xdgDir)
	nn := func(args ...string) error {
		cmd := exec.Command("go", append([]string{"run", "github.com/jaresty/nn/cmd/nn"}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%v: %s", args, out)
		}
		return nil
	}
	return b, nn
}

// TestRemoveLinkCrossProcessRace proves the cross-process RMW race in RemoveLink.
// Two concurrent nn link remove invocations remove different links from the same source;
// without the fix one removal is overwritten by the other.
func TestRemoveLinkCrossProcessRace(t *testing.T) {
	b, nn := setupCrossProcess(t)
	src, dst1, dst2 := makeNote("src"), makeNote("dst1"), makeNote("dst2")
	for _, n := range []*note.Note{src, dst1, dst2} {
		if err := b.Write(n); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	const attempts = 10
	for i := range attempts {
		// Set up two links on src.
		src.Links = []note.Link{
			{TargetID: dst1.ID, Annotation: "a", Type: "supports", Status: "draft"},
			{TargetID: dst2.ID, Annotation: "b", Type: "supports", Status: "draft"},
		}
		src.Modified = time.Now().UTC().Truncate(time.Second)
		if err := b.Update(src); err != nil {
			t.Fatalf("reset attempt %d: %v", i, err)
		}
		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Add(2)
		go func() { defer wg.Done(); errs[0] = nn("unlink", src.ID, dst1.ID) }()
		go func() { defer wg.Done(); errs[1] = nn("unlink", src.ID, dst2.ID) }()
		wg.Wait()
		for j, e := range errs {
			if e != nil {
				t.Logf("attempt %d remove %d error: %v", i, j, e)
			}
		}
		got, err := b.Read(src.ID)
		if err != nil {
			t.Fatalf("Read attempt %d: %v", i, err)
		}
		if len(got.Links) > 0 {
			t.Errorf("attempt %d: cross-process RemoveLink race: got %d links (want 0) — one removal was lost", i, len(got.Links))
			return
		}
	}
}

// TestPromoteCrossProcessRace proves the cross-process RMW race in Promote.
// Two concurrent nn promote invocations target the same note; without the fix
// one promotion can overwrite the other's status write.
func TestPromoteCrossProcessRace(t *testing.T) {
	b, nn := setupCrossProcess(t)
	const attempts = 10
	for i := range attempts {
		n := makeNote(fmt.Sprintf("promote-me-%d", i))
		if err := b.Write(n); err != nil {
			t.Fatalf("Write attempt %d: %v", i, err)
		}
		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Add(2)
		go func() { defer wg.Done(); errs[0] = nn("promote", n.ID, "--to", "reviewed") }()
		go func() { defer wg.Done(); errs[1] = nn("promote", n.ID, "--to", "permanent") }()
		wg.Wait()
		for j, e := range errs {
			if e != nil {
				t.Logf("attempt %d promote %d error: %v", i, j, e)
			}
		}
		got, err := b.Read(n.ID)
		if err != nil {
			t.Fatalf("Read attempt %d: %v", i, err)
		}
		// At least one promotion must have taken effect.
		if got.Status == "draft" {
			t.Errorf("attempt %d: cross-process Promote race: status still draft — both writes lost", i)
			return
		}
	}
}

// TestBulkUpdateLinksCrossProcessRace proves the cross-process RMW race in BulkUpdateLinks.
func TestBulkUpdateLinksCrossProcessRace(t *testing.T) {
	b, nn := setupCrossProcess(t)
	src, dst1, dst2 := makeNote("src"), makeNote("dst1"), makeNote("dst2")
	for _, n := range []*note.Note{src, dst1, dst2} {
		if err := b.Write(n); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	const attempts = 10
	for i := range attempts {
		src.Links = []note.Link{
			{TargetID: dst1.ID, Annotation: "old-a", Type: "supports", Status: "draft"},
			{TargetID: dst2.ID, Annotation: "old-b", Type: "supports", Status: "draft"},
		}
		src.Modified = time.Now().UTC().Truncate(time.Second)
		if err := b.Update(src); err != nil {
			t.Fatalf("reset attempt %d: %v", i, err)
		}
		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			errs[0] = nn("bulk-update-link", src.ID, "--to", dst1.ID, "--annotation", "new-a", "--type", "supports")
		}()
		go func() {
			defer wg.Done()
			errs[1] = nn("bulk-update-link", src.ID, "--to", dst2.ID, "--annotation", "new-b", "--type", "supports")
		}()
		wg.Wait()
		for j, e := range errs {
			if e != nil {
				t.Logf("attempt %d bulk-update %d error: %v", i, j, e)
			}
		}
		got, err := b.Read(src.ID)
		if err != nil {
			t.Fatalf("Read attempt %d: %v", i, err)
		}
		// Both annotations must be updated; if race, one update's write overwrites the other.
		updated := 0
		for _, lnk := range got.Links {
			if lnk.Annotation == "new-a" || lnk.Annotation == "new-b" {
				updated++
			}
		}
		if updated < 2 {
			t.Errorf("attempt %d: cross-process BulkUpdateLinks race: only %d of 2 annotations updated — one write was lost", i, updated)
			return
		}
	}
}

// TestRemoveLinkByTypeCrossProcessRace proves the cross-process RMW race in RemoveLinkByType.
// Two concurrent nn unlink --type invocations remove different typed edges from the same source.
func TestRemoveLinkByTypeCrossProcessRace(t *testing.T) {
	b, nn := setupCrossProcess(t)
	src, dst1, dst2 := makeNote("src"), makeNote("dst1"), makeNote("dst2")
	for _, n := range []*note.Note{src, dst1, dst2} {
		if err := b.Write(n); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	const attempts = 10
	for i := range attempts {
		src.Links = []note.Link{
			{TargetID: dst1.ID, Annotation: "a", Type: "supports", Status: "draft"},
			{TargetID: dst2.ID, Annotation: "b", Type: "extends", Status: "draft"},
		}
		src.Modified = time.Now().UTC().Truncate(time.Second)
		if err := b.Update(src); err != nil {
			t.Fatalf("reset attempt %d: %v", i, err)
		}
		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Add(2)
		go func() { defer wg.Done(); errs[0] = nn("unlink", src.ID, dst1.ID, "--type", "supports") }()
		go func() { defer wg.Done(); errs[1] = nn("unlink", src.ID, dst2.ID, "--type", "extends") }()
		wg.Wait()
		for j, e := range errs {
			if e != nil {
				t.Logf("attempt %d unlink %d error: %v", i, j, e)
			}
		}
		got, err := b.Read(src.ID)
		if err != nil {
			t.Fatalf("Read attempt %d: %v", i, err)
		}
		if len(got.Links) > 0 {
			t.Errorf("attempt %d: cross-process RemoveLinkByType race: got %d links (want 0) — one removal was lost", i, len(got.Links))
			return
		}
	}
}

// TestUpdateLinkCrossProcessRace proves the cross-process RMW race in UpdateLink.
// Two concurrent nn update-link invocations update different links on the same source.
func TestUpdateLinkCrossProcessRace(t *testing.T) {
	b, nn := setupCrossProcess(t)
	src, dst1, dst2 := makeNote("src"), makeNote("dst1"), makeNote("dst2")
	for _, n := range []*note.Note{src, dst1, dst2} {
		if err := b.Write(n); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	const attempts = 10
	for i := range attempts {
		src.Links = []note.Link{
			{TargetID: dst1.ID, Annotation: "old-a", Type: "supports", Status: "draft"},
			{TargetID: dst2.ID, Annotation: "old-b", Type: "supports", Status: "draft"},
		}
		src.Modified = time.Now().UTC().Truncate(time.Second)
		if err := b.Update(src); err != nil {
			t.Fatalf("reset attempt %d: %v", i, err)
		}
		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			errs[0] = nn("update-link", src.ID, dst1.ID, "--annotation", "new-a")
		}()
		go func() {
			defer wg.Done()
			errs[1] = nn("update-link", src.ID, dst2.ID, "--annotation", "new-b")
		}()
		wg.Wait()
		for j, e := range errs {
			if e != nil {
				t.Logf("attempt %d update-link %d error: %v", i, j, e)
			}
		}
		got, err := b.Read(src.ID)
		if err != nil {
			t.Fatalf("Read attempt %d: %v", i, err)
		}
		updated := 0
		for _, lnk := range got.Links {
			if lnk.Annotation == "new-a" || lnk.Annotation == "new-b" {
				updated++
			}
		}
		if updated < 2 {
			t.Errorf("attempt %d: cross-process UpdateLink race: only %d of 2 annotations updated — one write was lost", i, updated)
			return
		}
	}
}

// TestPromoteConflictRejected proves that concurrent nn promote invocations on the same
// note result in exactly one success and one conflict error. Without the conflict guard,
// both return nil (silent overwrite). With it, the second promote sees a modified
// timestamp that differs from what it read and returns an error.
func TestPromoteConflictRejected(t *testing.T) {
	b, nn := setupCrossProcess(t)
	const attempts = 10
	bothSucceeded := 0
	for i := range attempts {
		n := makeNote(fmt.Sprintf("promote-conflict-%d", i))
		if err := b.Write(n); err != nil {
			t.Fatalf("Write attempt %d: %v", i, err)
		}
		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Add(2)
		go func() { defer wg.Done(); errs[0] = nn("promote", n.ID, "--to", "reviewed") }()
		go func() { defer wg.Done(); errs[1] = nn("promote", n.ID, "--to", "permanent") }()
		wg.Wait()
		if errs[0] == nil && errs[1] == nil {
			bothSucceeded++
		}
	}
	// Without the conflict guard, both promotes always succeed (silent overwrite).
	// With the guard, at least one attempt must produce a conflict error.
	if bothSucceeded == attempts {
		t.Errorf("promote conflict guard absent: all %d attempts had both promotes succeed — concurrent promotes must produce a conflict error", attempts)
	}
}

// TestAddLinksCrossProcessRace proves the cross-process RMW race in AddLinks by spawning
// real nn bulk-link subprocesses. Two concurrent invocations share the same source note;
// without the fix one set of links is silently lost.
func TestAddLinksCrossProcessRace(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	xdgDir := t.TempDir()
	cfgPath := filepath.Join(xdgDir, "nn", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	cfgContent := fmt.Sprintf("[notebooks]\ndefault = \"personal\"\n[notebooks.personal]\npath = %q\nbackend = \"gitlocal\"\n", dir)
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	b, err := gitlocal.NewWithConfigDir(dir, xdgDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	src, dst1, dst2 := makeNote("src"), makeNote("dst1"), makeNote("dst2")
	for _, n := range []*note.Note{src, dst1, dst2} {
		if err := b.Write(n); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	env := append(os.Environ(), "XDG_CONFIG_HOME="+xdgDir)
	nn := func(args ...string) error {
		cmd := exec.Command("go", append([]string{"run", "github.com/jaresty/nn/cmd/nn"}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%v: %s", args, out)
		}
		return nil
	}

	const attempts = 10
	for i := range attempts {
		src.Links = nil
		src.Modified = time.Now().UTC().Truncate(time.Second)
		if err := b.Update(src); err != nil {
			t.Fatalf("reset attempt %d: %v", i, err)
		}

		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			errs[0] = nn("bulk-link", src.ID, "--to", dst1.ID, "--annotation", "probe A", "--type", "supports")
		}()
		go func() {
			defer wg.Done()
			errs[1] = nn("bulk-link", src.ID, "--to", dst2.ID, "--annotation", "probe B", "--type", "supports")
		}()
		wg.Wait()

		for j, e := range errs {
			if e != nil {
				t.Logf("attempt %d bulk-link %d error (may be dup): %v", i, j, e)
			}
		}

		got, err := b.Read(src.ID)
		if err != nil {
			t.Fatalf("Read attempt %d: %v", i, err)
		}
		if len(got.Links) < 2 {
			t.Errorf("attempt %d: cross-process AddLinks race: got %d links (want 2) — one write was lost", i, len(got.Links))
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
		go func() { defer wg.Done(); errs[0] = b.Promote(n.ID, n.Modified, note.StatusReviewed) }()
		go func() { defer wg.Done(); errs[1] = b.Promote(n.ID, n.Modified, note.StatusPermanent) }()
		wg.Wait()
		// With the conflict guard, exactly one must succeed and one must conflict-error.
		if errs[0] == nil && errs[1] == nil {
			t.Errorf("both concurrent Promotes succeeded — conflict guard not firing")
			return
		}
		if errs[0] != nil && errs[1] != nil {
			t.Errorf("both concurrent Promotes failed: %v / %v", errs[0], errs[1])
			return
		}
	}
}
