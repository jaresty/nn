package gitlocal_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/backend/gitlocal"
	"github.com/jaresty/nn/internal/note"
)

// TestEachWriteProducesOwnCommit verifies that concurrent Write calls each produce
// their own git commit containing only their own file — not a sweep commit.
func TestEachWriteProducesOwnCommit(t *testing.T) {
	dir := t.TempDir()
	configDir := t.TempDir()
	initGitRepo(t, dir)
	b, err := gitlocal.NewWithConfigDir(dir, configDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	const n = 5
	var wg sync.WaitGroup
	errs := make([]error, n)
	ids := make([]string, n)
	for i := range n {
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
				Body:     fmt.Sprintf("body %d", idx),
			}
			ids[idx] = nn.ID
			errs[idx] = b.Write(nn)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("Write %d failed: %v", i, err)
		}
	}

	// Each commit should touch exactly one file.
	// Skip the initial "init" commit by looking only at commits that mention one of our note IDs.
	cmd := exec.Command("git", "log", "--name-only", "--format=%H")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log: %v\n%s", err, out)
	}
	// Parse blocks: each commit hash is followed by the files it changed.
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	commitFiles := map[string][]string{}
	var currentHash string
	for _, line := range lines {
		if line == "" {
			continue
		}
		if len(line) == 40 { // SHA
			currentHash = line
		} else if strings.HasSuffix(line, ".md") {
			commitFiles[currentHash] = append(commitFiles[currentHash], line)
		}
	}
	// Filter to commits that touched one of our concurrently-written notes.
	noteSet := map[string]bool{}
	for _, id := range ids {
		noteSet[id] = true
	}
	for hash, files := range commitFiles {
		// Only check commits that include at least one of our notes.
		relevant := false
		for _, f := range files {
			for id := range noteSet {
				if strings.Contains(f, id) {
					relevant = true
				}
			}
		}
		if !relevant {
			continue
		}
		if len(files) != 1 {
			t.Errorf("commit %s touched %d files (want 1): %v", hash, len(files), files)
		}
	}
}

// TestCrossProcessWritesEachProduceOwnCommit spawns N separate nn processes
// concurrently via go run. Each process creates one note. The test asserts that
// every git commit touches exactly one file — verifying that git commit --only
// prevents one process from sweeping another's staged file.
func TestCrossProcessWritesEachProduceOwnCommit(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	// Write a minimal config TOML under a temp XDG_CONFIG_HOME so nn resolves
	// the notebook without touching the real user config.
	xdgDir := t.TempDir()
	cfgPath := filepath.Join(xdgDir, "nn", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	cfgContent := fmt.Sprintf("[notebooks]\ndefault = \"personal\"\n[notebooks.personal]\npath = %q\nbackend = \"gitlocal\"\n", dir)
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	const n = 5
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cmd := exec.Command("go", "run", "github.com/jaresty/nn/cmd/nn",
				"new", "--quick",
				"--title", fmt.Sprintf("Cross-Process Note %d", idx),
				"--no-edit",
			)
			cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+xdgDir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				errs[idx] = fmt.Errorf("nn new %d: %v\n%s", idx, err, out)
			}
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("process %d failed: %v", i, err)
		}
	}

	// Each commit should touch exactly one file.
	log := exec.Command("git", "log", "--name-only", "--format=%H")
	log.Dir = dir
	out, err := log.CombinedOutput()
	if err != nil {
		t.Fatalf("git log: %v\n%s", err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	commitFiles := map[string][]string{}
	var currentHash string
	for _, line := range lines {
		if line == "" {
			continue
		}
		if len(line) == 40 {
			currentHash = line
		} else if strings.HasSuffix(line, ".md") {
			commitFiles[currentHash] = append(commitFiles[currentHash], line)
		}
	}
	for hash, files := range commitFiles {
		if len(files) != 1 {
			t.Errorf("commit %s touched %d files (want 1): %v", hash, len(files), files)
		}
	}
}

// TestCrossProcessPromotesNoIndexLock spawns N nn promote processes concurrently
// and asserts that none fail with an index.lock error.
func TestCrossProcessPromotesNoIndexLock(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	xdgDir := t.TempDir()
	cfgPath := filepath.Join(xdgDir, "nn", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	cfgContent := fmt.Sprintf("[notebooks]\ndefault = \"personal\"\n[notebooks.personal]\npath = %q\nbackend = \"gitlocal\"\n", dir)
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Pre-create notes serially so we have IDs to promote.
	const n = 5
	ids := make([]string, n)
	b, err := gitlocal.NewWithConfigDir(dir, filepath.Join(xdgDir, "nn"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for i := range n {
		nn := &note.Note{
			ID:       note.GenerateID(),
			Title:    fmt.Sprintf("Promote Target %d", i),
			Type:     note.TypeObservation,
			Status:   note.StatusDraft,
			Created:  time.Now().UTC().Truncate(time.Second),
			Modified: time.Now().UTC().Truncate(time.Second),
		}
		if err := b.Write(nn); err != nil {
			t.Fatalf("setup Write %d: %v", i, err)
		}
		ids[i] = nn.ID
	}

	// Promote all notes concurrently via separate nn processes.
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cmd := exec.Command("go", "run", "github.com/jaresty/nn/cmd/nn",
				"promote", ids[idx], "--to", "reviewed",
			)
			cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+xdgDir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				errs[idx] = fmt.Errorf("nn promote %d: %v\n%s", idx, err, out)
			}
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("process %d failed: %v", i, err)
		}
	}
}

// TestParallelUpdates verifies that concurrent Write calls on the same backend
// all succeed — none fail due to git index.lock contention.
func TestParallelUpdates(t *testing.T) {
	dir := t.TempDir()
	configDir := t.TempDir()
	initGitRepo(t, dir)
	b, err := gitlocal.NewWithConfigDir(dir, configDir)
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
			errs[idx] = b.Update(nn, nil)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("parallel Update %d failed: %v", i, err)
		}
	}
}
