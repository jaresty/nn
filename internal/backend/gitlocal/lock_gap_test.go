package gitlocal_test

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/backend/gitlocal"
	"github.com/jaresty/nn/internal/note"
)

// TestBulkWriteConcurrent verifies that concurrent BulkWrite calls each produce
// their own commit containing exactly their own files — no sweep commits.
func TestBulkWriteConcurrent(t *testing.T) {
	dir := t.TempDir()
	configDir := t.TempDir()
	initGitRepo(t, dir)
	b, err := gitlocal.NewWithConfigDir(dir, configDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	const batches = 4
	const notesPerBatch = 2
	var wg sync.WaitGroup
	errs := make([]error, batches)
	batchIDs := make([][]string, batches)
	for i := range batches {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			notes := make([]*note.Note, notesPerBatch)
			batchIDs[idx] = make([]string, notesPerBatch)
			for j := range notesPerBatch {
				notes[j] = &note.Note{
					ID:       note.GenerateID(),
					Title:    fmt.Sprintf("Bulk Note %d-%d", idx, j),
					Type:     note.TypeConcept,
					Status:   note.StatusDraft,
					Created:  time.Now().UTC().Truncate(time.Second),
					Modified: time.Now().UTC().Truncate(time.Second),
				}
				batchIDs[idx][j] = notes[j].ID
			}
			errs[idx] = b.BulkWrite(notes)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("BulkWrite %d failed: %v", i, err)
		}
	}

	// Parse git log: each bulk-new commit must touch exactly notesPerBatch files.
	cmd := exec.Command("git", "log", "--name-only", "--format=%H %s")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log: %v\n%s", err, out)
	}
	// git log --name-only format: "<hash subject>\n\n<file>\n<file>\n\n<hash subject>..."
	// Group by commit: split on lines that look like "<40-char-hash> <subject>".
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	commitFiles := map[string][]string{}
	var currentHash string
	for _, line := range lines {
		if line == "" {
			continue
		}
		if len(line) > 40 && line[40] == ' ' {
			currentHash = line
		} else if strings.HasSuffix(line, ".md") {
			commitFiles[currentHash] = append(commitFiles[currentHash], line)
		}
	}
	for hash, files := range commitFiles {
		if strings.Contains(hash, "bulk-new") && len(files) != notesPerBatch {
			t.Errorf("commit %q touched %d files (want %d): %v", hash, len(files), notesPerBatch, files)
		}
	}
}

// TestUpdateRenameHoldsLock verifies that renaming a note (title change causing
// slug change) does not race with concurrent writes — the git rm --cached and
// git add+commit must all be serialized under one lock acquisition.
func TestUpdateRenameHoldsLock(t *testing.T) {
	dir := t.TempDir()
	configDir := t.TempDir()
	initGitRepo(t, dir)
	b, err := gitlocal.NewWithConfigDir(dir, configDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Create a note to rename.
	n := &note.Note{
		ID:       note.GenerateID(),
		Title:    "Original Title",
		Type:     note.TypeConcept,
		Status:   note.StatusDraft,
		Created:  time.Now().UTC().Truncate(time.Second),
		Modified: time.Now().UTC().Truncate(time.Second),
	}
	if err := b.Write(n); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Concurrently rename the note while writing others.
	const writers = 4
	var wg sync.WaitGroup
	errs := make([]error, writers+1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		n.Title = "Renamed Title"
		n.Modified = time.Now().UTC().Truncate(time.Second)
		errs[0] = b.Update(n, nil)
	}()

	for i := range writers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			nn := &note.Note{
				ID:       note.GenerateID(),
				Title:    fmt.Sprintf("Concurrent Note %d", idx),
				Type:     note.TypeConcept,
				Status:   note.StatusDraft,
				Created:  time.Now().UTC().Truncate(time.Second),
				Modified: time.Now().UTC().Truncate(time.Second),
			}
			errs[idx+1] = b.Write(nn)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d failed: %v", i, err)
		}
	}
}
