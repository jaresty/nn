// Package gitlocal implements the Backend interface using the local filesystem
// with a Git repository for history. Each write operation produces one commit.
package gitlocal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/jaresty/nn/internal/backend"
	"github.com/jaresty/nn/internal/note"
)

const listWorkers = 16

// readFilesConcurrently reads each path using a bounded worker pool and returns
// the byte slices in the same order as the input slice. Each goroutine writes
// to a distinct index so no mutex is needed on results.
func readFilesConcurrently(paths []string) ([][]byte, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	results := make([][]byte, len(paths))
	g := new(errgroup.Group)
	sem := make(chan struct{}, listWorkers)
	for i, p := range paths {
		sem <- struct{}{}
		g.Go(func() error {
			defer func() { <-sem }()
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			results[i] = data
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

// atomicWriteFile writes data to path via a temp file + rename so concurrent
// readers never observe a partially-written file (avoids O_TRUNC visibility window on Linux).
func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".nn-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

// Backend stores notes as Markdown files in a Git-backed directory.
// mu serialises add+commit pairs so concurrent goroutines in the same process
// do not interleave staging. Cross-process isolation is handled by the commit queue.
type Backend struct {
	dir       string
	configDir string // directory holding the commit queue and lock file
	mu        sync.Mutex
}

// New returns a Backend rooted at dir, using the default nn config directory
// for the commit queue.
func New(dir string) (*Backend, error) {
	return newBackendWithConfigDir(dir, defaultNNConfigDir())
}

// NewWithConfigDir returns a Backend rooted at dir using configDir for the
// commit queue. Used in tests to isolate queue state.
func NewWithConfigDir(dir, configDir string) (*Backend, error) {
	return newBackendWithConfigDir(dir, configDir)
}

func newBackendWithConfigDir(dir, configDir string) (*Backend, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("gitlocal.New: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("gitlocal.New: %q is not a directory", dir)
	}
	return &Backend{dir: dir, configDir: configDir}, nil
}

func defaultNNConfigDir() string {
	if d := os.Getenv("NN_CONFIG_DIR"); d != "" {
		return d
	}
	cfgDir := os.Getenv("XDG_CONFIG_HOME")
	if cfgDir == "" {
		home, _ := os.UserHomeDir()
		cfgDir = filepath.Join(home, ".config")
	}
	return filepath.Join(cfgDir, "nn")
}

// Write serialises n to a Markdown file and commits it to Git.
func (b *Backend) Write(n *note.Note) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for {
		if _, err := b.findByID(n.ID); err != nil {
			break // no collision
		}
		n.ID = note.GenerateID()
	}
	data, err := n.Marshal()
	if err != nil {
		return fmt.Errorf("gitlocal.Write: %w", err)
	}
	path := filepath.Join(b.dir, n.Filename())
	if err := atomicWriteFile(path, data); err != nil {
		return fmt.Errorf("gitlocal.Write: %w", err)
	}
	msg := fmt.Sprintf("note: create %s — %s", n.ID, n.Title)
	return b.commitLocked(path, msg)
}

// Read finds and parses the note with the given ID.
func (b *Backend) Read(id string) (*note.Note, error) {
	path, err := b.findByID(id)
	if err != nil {
		return nil, fmt.Errorf("gitlocal.Read: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("gitlocal.Read: %w", err)
	}
	n, err := note.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("gitlocal.Read: %w", err)
	}
	return n, nil
}

// Delete removes the note file for id and commits the deletion.
func (b *Backend) Delete(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	path, err := b.findByID(id)
	if err != nil {
		return fmt.Errorf("gitlocal.Delete: %w", err)
	}
	// Read title for the commit message before deleting.
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("gitlocal.Delete: read: %w", err)
	}
	n, err := note.Parse(data)
	if err != nil {
		return fmt.Errorf("gitlocal.Delete: parse: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("gitlocal.Delete: remove: %w", err)
	}
	// Cascade: remove edges pointing to the deleted note from all other notes.
	if err := b.removeInboundEdgesLocked(n.ID); err != nil {
		return fmt.Errorf("gitlocal.Delete: cascade: %w", err)
	}
	msg := fmt.Sprintf("note: delete %s — %s", n.ID, n.Title)
	return b.commitDeleteLocked(path, msg)
}

// List returns all notes in the notebook directory.
func (b *Backend) List() ([]*note.Note, error) {
	entries, err := os.ReadDir(b.dir)
	if err != nil {
		return nil, fmt.Errorf("gitlocal.List: %w", err)
	}
	var mdNames []string
	var mdPaths []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		mdNames = append(mdNames, e.Name())
		mdPaths = append(mdPaths, filepath.Join(b.dir, e.Name()))
	}
	contents, err := readFilesConcurrently(mdPaths)
	if err != nil {
		if os.IsNotExist(err) {
			// A file disappeared between ReadDir and ReadFile — skip silently.
			// Fall back to sequential read to find which files are still present.
			var notes []*note.Note
			for i, p := range mdPaths {
				data, rerr := os.ReadFile(p)
				if os.IsNotExist(rerr) {
					continue
				}
				if rerr != nil {
					return nil, fmt.Errorf("gitlocal.List: read %s: %w", mdNames[i], rerr)
				}
				n, perr := note.Parse(data)
				if perr != nil {
					continue
				}
				notes = append(notes, n)
			}
			return notes, nil
		}
		return nil, fmt.Errorf("gitlocal.List: %w", err)
	}
	var notes []*note.Note
	for _, data := range contents {
		n, perr := note.Parse(data)
		if perr != nil {
			continue
		}
		notes = append(notes, n)
	}
	return notes, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// findByID locates the file whose name begins with id.
func (b *Backend) findByID(id string) (string, error) {
	entries, err := os.ReadDir(b.dir)
	if err != nil {
		return "", fmt.Errorf("findByID: %w", err)
	}
	prefix := id + "-"
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".md") {
			return filepath.Join(b.dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("note %q not found", id)
}

// commitLocked runs git add+commit under the cross-process git lock.
// Caller must hold b.mu.
func (b *Backend) commitLocked(path, msg string) error {
	if err := acquireGitLock(b.configDir); err != nil {
		return fmt.Errorf("gitlocal.commit: %w", err)
	}
	defer releaseGitLock(b.configDir)
	if out, err := gitCmdIn(b.dir, "add", path); err != nil {
		return fmt.Errorf("gitlocal.commit: git add: %w\n%s", err, out)
	}
	if nothingStaged(b.dir) {
		return nil
	}
	if out, err := gitCmdIn(b.dir, "commit", "-m", msg); err != nil {
		return fmt.Errorf("gitlocal.commit: git commit: %w\n%s", err, out)
	}
	return nil
}

// commitWithLockHeld runs git add+commit. Caller must hold both b.mu and the git lock.
func (b *Backend) commitWithLockHeld(path, msg string) error {
	if out, err := gitCmdIn(b.dir, "add", path); err != nil {
		return fmt.Errorf("gitlocal.commit: git add: %w\n%s", err, out)
	}
	if nothingStaged(b.dir) {
		return nil
	}
	if out, err := gitCmdIn(b.dir, "commit", "-m", msg); err != nil {
		return fmt.Errorf("gitlocal.commit: git commit: %w\n%s", err, out)
	}
	return nil
}

// commitDeleteLocked runs git rm+commit under the cross-process git lock.
// Caller must hold b.mu.
func (b *Backend) commitDeleteLocked(path, msg string) error {
	if err := acquireGitLock(b.configDir); err != nil {
		return fmt.Errorf("gitlocal.commitDelete: %w", err)
	}
	defer releaseGitLock(b.configDir)
	if out, err := gitCmdIn(b.dir, "rm", "--cached", "--ignore-unmatch", path); err != nil {
		return fmt.Errorf("gitlocal.commitDelete: git rm: %w\n%s", err, out)
	}
	if nothingStaged(b.dir) {
		return nil
	}
	if out, err := gitCmdIn(b.dir, "commit", "-m", msg); err != nil {
		return fmt.Errorf("gitlocal.commitDelete: git commit: %w\n%s", err, out)
	}
	return nil
}

// commitRenameLocked runs git rm+add+commit under the cross-process git lock.
// Caller must hold b.mu.
func (b *Backend) commitRenameLocked(oldPath, newPath, msg string) error {
	if err := acquireGitLock(b.configDir); err != nil {
		return fmt.Errorf("gitlocal.commitRename: %w", err)
	}
	defer releaseGitLock(b.configDir)
	if out, err := gitCmdIn(b.dir, "rm", "--cached", "--ignore-unmatch", oldPath); err != nil {
		return fmt.Errorf("gitlocal.commitRename: git rm: %w\n%s", err, out)
	}
	if out, err := gitCmdIn(b.dir, "add", newPath); err != nil {
		return fmt.Errorf("gitlocal.commitRename: git add: %w\n%s", err, out)
	}
	if out, err := gitCmdIn(b.dir, "commit", "-m", msg); err != nil {
		return fmt.Errorf("gitlocal.commitRename: git commit: %w\n%s", err, out)
	}
	return nil
}

// commitBulkLocked runs git add (all paths)+commit under the cross-process git lock.
// Caller must hold b.mu.
func (b *Backend) commitBulkLocked(paths []string, msg string) error {
	if err := acquireGitLock(b.configDir); err != nil {
		return fmt.Errorf("gitlocal.commitBulk: %w", err)
	}
	defer releaseGitLock(b.configDir)
	addArgs := append([]string{"add"}, paths...)
	if out, err := gitCmdIn(b.dir, addArgs...); err != nil {
		return fmt.Errorf("gitlocal.commitBulk: git add: %w\n%s", err, out)
	}
	if out, err := gitCmdIn(b.dir, "commit", "-m", msg); err != nil {
		return fmt.Errorf("gitlocal.commitBulk: git commit: %w\n%s", err, out)
	}
	return nil
}


// AddLink adds an annotated link from fromID to toID and commits.
func (b *Backend) AddLink(fromID, toID, annotation, linkType, linkStatus string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := acquireGitLock(b.configDir); err != nil {
		return fmt.Errorf("gitlocal.AddLink: %w", err)
	}
	defer releaseGitLock(b.configDir)
	n, err := b.Read(fromID)
	if err != nil {
		return fmt.Errorf("gitlocal.AddLink: %w", err)
	}
	for _, lnk := range n.Links {
		if lnk.TargetID == toID && lnk.Type == linkType {
			return fmt.Errorf("gitlocal.AddLink: link %s→%s [%s] already exists", fromID, toID, linkType)
		}
	}
	n.Links = append(n.Links, note.Link{TargetID: toID, Annotation: annotation, Type: linkType, Status: linkStatus})
	data, err := n.Marshal()
	if err != nil {
		return fmt.Errorf("gitlocal.AddLink: marshal: %w", err)
	}
	path := filepath.Join(b.dir, n.Filename())
	if err := atomicWriteFile(path, data); err != nil {
		return fmt.Errorf("gitlocal.AddLink: write: %w", err)
	}
	msg := fmt.Sprintf("note: link %s → %s", fromID, toID)
	return b.commitWithLockHeld(path, msg)
}

// AddLinks adds multiple annotated links from fromID in a single git commit.
func (b *Backend) AddLinks(fromID string, targets []backend.LinkTarget) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := acquireGitLock(b.configDir); err != nil {
		return fmt.Errorf("gitlocal.AddLinks: %w", err)
	}
	defer releaseGitLock(b.configDir)
	n, err := b.Read(fromID)
	if err != nil {
		return fmt.Errorf("gitlocal.AddLinks: %w", err)
	}
	type edgeKey struct{ toID, linkType string }
	existing := make(map[edgeKey]bool, len(n.Links))
	for _, lnk := range n.Links {
		existing[edgeKey{lnk.TargetID, lnk.Type}] = true
	}
	for _, t := range targets {
		k := edgeKey{t.ToID, t.Type}
		if existing[k] {
			return fmt.Errorf("gitlocal.AddLinks: link %s→%s [%s] already exists", fromID, t.ToID, t.Type)
		}
		n.Links = append(n.Links, note.Link{TargetID: t.ToID, Annotation: t.Annotation, Type: t.Type, Status: t.Status})
		existing[k] = true
	}
	data, err := n.Marshal()
	if err != nil {
		return fmt.Errorf("gitlocal.AddLinks: marshal: %w", err)
	}
	path := filepath.Join(b.dir, n.Filename())
	if err := atomicWriteFile(path, data); err != nil {
		return fmt.Errorf("gitlocal.AddLinks: write: %w", err)
	}
	msg := fmt.Sprintf("note: bulk-link %s → %d notes", fromID, len(targets))
	return b.commitWithLockHeld(path, msg)
}

// removeInboundEdgesLocked removes all links targeting deletedID from every other note.
// Caller must hold b.mu.
func (b *Backend) removeInboundEdgesLocked(deletedID string) error {
	notes, err := b.List()
	if err != nil {
		return err
	}
	for _, n := range notes {
		var keep []note.Link
		for _, lnk := range n.Links {
			if lnk.TargetID != deletedID {
				keep = append(keep, lnk)
			}
		}
		if len(keep) == len(n.Links) {
			continue
		}
		n.Links = keep
		data, err := n.Marshal()
		if err != nil {
			return fmt.Errorf("removeInboundEdges: marshal %s: %w", n.ID, err)
		}
		p := filepath.Join(b.dir, n.Filename())
		if err := atomicWriteFile(p, data); err != nil {
			return fmt.Errorf("removeInboundEdges: write %s: %w", n.ID, err)
		}
		msg := fmt.Sprintf("note: unlink %s → %s (cascade delete)", n.ID, deletedID)
		if err := b.commitLocked(p, msg); err != nil {
			return err
		}
	}
	return nil
}

// RemoveLink removes the link from fromID to toID and commits.
func (b *Backend) RemoveLink(fromID, toID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := acquireGitLock(b.configDir); err != nil {
		return fmt.Errorf("gitlocal.RemoveLink: %w", err)
	}
	defer releaseGitLock(b.configDir)
	n, err := b.Read(fromID)
	if err != nil {
		return fmt.Errorf("gitlocal.RemoveLink: %w", err)
	}
	filtered := n.Links[:0]
	for _, lnk := range n.Links {
		if lnk.TargetID != toID {
			filtered = append(filtered, lnk)
		}
	}
	n.Links = filtered
	data, err := n.Marshal()
	if err != nil {
		return fmt.Errorf("gitlocal.RemoveLink: marshal: %w", err)
	}
	path := filepath.Join(b.dir, n.Filename())
	if err := atomicWriteFile(path, data); err != nil {
		return fmt.Errorf("gitlocal.RemoveLink: write: %w", err)
	}
	msg := fmt.Sprintf("note: unlink %s → %s", fromID, toID)
	return b.commitWithLockHeld(path, msg)
}

// RemoveLinkByType removes only edges from fromID to toID with the given type.
func (b *Backend) RemoveLinkByType(fromID, toID, linkType string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := acquireGitLock(b.configDir); err != nil {
		return fmt.Errorf("gitlocal.RemoveLinkByType: %w", err)
	}
	defer releaseGitLock(b.configDir)
	n, err := b.Read(fromID)
	if err != nil {
		return fmt.Errorf("gitlocal.RemoveLinkByType: %w", err)
	}
	filtered := n.Links[:0]
	for _, lnk := range n.Links {
		if !(lnk.TargetID == toID && lnk.Type == linkType) {
			filtered = append(filtered, lnk)
		}
	}
	n.Links = filtered
	data, err := n.Marshal()
	if err != nil {
		return fmt.Errorf("gitlocal.RemoveLinkByType: marshal: %w", err)
	}
	path := filepath.Join(b.dir, n.Filename())
	if err := atomicWriteFile(path, data); err != nil {
		return fmt.Errorf("gitlocal.RemoveLinkByType: write: %w", err)
	}
	msg := fmt.Sprintf("note: unlink %s → %s [%s]", fromID, toID, linkType)
	return b.commitWithLockHeld(path, msg)
}

// BulkUpdateLinks applies multiple link updates to fromID in a single git commit.
func (b *Backend) BulkUpdateLinks(fromID string, updates []backend.LinkUpdate) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := acquireGitLock(b.configDir); err != nil {
		return fmt.Errorf("gitlocal.BulkUpdateLinks: %w", err)
	}
	defer releaseGitLock(b.configDir)
	n, err := b.Read(fromID)
	if err != nil {
		return fmt.Errorf("gitlocal.BulkUpdateLinks: %w", err)
	}
	for _, u := range updates {
		found := false
		for i, lnk := range n.Links {
			if lnk.TargetID != u.ToID {
				continue
			}
			found = true
			if u.Annotation != nil {
				n.Links[i].Annotation = *u.Annotation
			}
			if u.Type != nil {
				n.Links[i].Type = *u.Type
			}
			if u.Status != nil {
				n.Links[i].Status = *u.Status
			}
			break
		}
		if !found {
			return fmt.Errorf("gitlocal.BulkUpdateLinks: link %s→%s not found", fromID, u.ToID)
		}
	}
	data, err := n.Marshal()
	if err != nil {
		return fmt.Errorf("gitlocal.BulkUpdateLinks: marshal: %w", err)
	}
	path := filepath.Join(b.dir, n.Filename())
	if err := atomicWriteFile(path, data); err != nil {
		return fmt.Errorf("gitlocal.BulkUpdateLinks: write: %w", err)
	}
	msg := fmt.Sprintf("note: bulk-update-link %s (%d links)", fromID, len(updates))
	return b.commitWithLockHeld(path, msg)
}

// UpdateLink modifies the annotation, type, and/or status of an existing link without removing it.
// nil pointer arguments mean "leave unchanged".
func (b *Backend) UpdateLink(fromID, toID string, annotation, linkType, linkStatus *string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := acquireGitLock(b.configDir); err != nil {
		return fmt.Errorf("gitlocal.UpdateLink: %w", err)
	}
	defer releaseGitLock(b.configDir)
	n, err := b.Read(fromID)
	if err != nil {
		return fmt.Errorf("gitlocal.UpdateLink: %w", err)
	}
	found := false
	for i, lnk := range n.Links {
		if lnk.TargetID != toID {
			continue
		}
		found = true
		if annotation != nil {
			n.Links[i].Annotation = *annotation
		}
		if linkType != nil {
			n.Links[i].Type = *linkType
		}
		if linkStatus != nil {
			n.Links[i].Status = *linkStatus
		}
		break
	}
	if !found {
		return fmt.Errorf("gitlocal.UpdateLink: link %s→%s not found", fromID, toID)
	}
	data, err := n.Marshal()
	if err != nil {
		return fmt.Errorf("gitlocal.UpdateLink: marshal: %w", err)
	}
	path := filepath.Join(b.dir, n.Filename())
	if err := atomicWriteFile(path, data); err != nil {
		return fmt.Errorf("gitlocal.UpdateLink: write: %w", err)
	}
	msg := fmt.Sprintf("note: update-link %s → %s", fromID, toID)
	return b.commitWithLockHeld(path, msg)
}

// Update writes the modified note and commits with an "update" message.
func (b *Backend) Update(n *note.Note) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	oldPath, err := b.findByID(n.ID)
	if err != nil {
		return fmt.Errorf("gitlocal.Update: note not found: %w", err)
	}
	data, err := n.Marshal()
	if err != nil {
		return fmt.Errorf("gitlocal.Update: %w", err)
	}
	newPath := filepath.Join(b.dir, n.Filename())
	if err := atomicWriteFile(newPath, data); err != nil {
		return fmt.Errorf("gitlocal.Update: %w", err)
	}
	msg := fmt.Sprintf("note: update %s — %s", n.ID, n.Title)
	if oldPath != newPath {
		// Title changed slug — remove old file; unstage+add+commit under one lock.
		if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("gitlocal.Update: remove old file: %w", err)
		}
		return b.commitRenameLocked(oldPath, newPath, msg)
	}
	return b.commitLocked(newPath, msg)
}

// Promote updates the status of the note with the given id and commits.
func (b *Backend) Promote(id string, from time.Time, to note.Status) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := acquireGitLock(b.configDir); err != nil {
		return fmt.Errorf("gitlocal.Promote: %w", err)
	}
	defer releaseGitLock(b.configDir)
	n, err := b.Read(id)
	if err != nil {
		return fmt.Errorf("gitlocal.Promote: %w", err)
	}
	if !n.Modified.Equal(from) {
		return fmt.Errorf("gitlocal.Promote: note %s was modified since you read it (conflict)", id)
	}
	n.Status = to
	n.Modified = time.Now().UTC()
	data, err := n.Marshal()
	if err != nil {
		return fmt.Errorf("gitlocal.Promote: marshal: %w", err)
	}
	path := filepath.Join(b.dir, n.Filename())
	if err := atomicWriteFile(path, data); err != nil {
		return fmt.Errorf("gitlocal.Promote: write: %w", err)
	}
	msg := fmt.Sprintf("note: promote %s to %s", id, string(to))
	return b.commitWithLockHeld(path, msg)
}

// BulkWrite writes all notes and commits in a single commit.
func (b *Backend) BulkWrite(notes []*note.Note) error {
	if len(notes) == 0 {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	var paths []string
	for _, n := range notes {
		for {
			if _, err := b.findByID(n.ID); err != nil {
				break // no collision
			}
			n.ID = note.GenerateID()
		}
		data, err := n.Marshal()
		if err != nil {
			return fmt.Errorf("gitlocal.BulkWrite: marshal %s: %w", n.ID, err)
		}
		path := filepath.Join(b.dir, n.Filename())
		if err := atomicWriteFile(path, data); err != nil {
			return fmt.Errorf("gitlocal.BulkWrite: write %s: %w", n.ID, err)
		}
		paths = append(paths, path)
	}
	return b.commitBulkLocked(paths, fmt.Sprintf("note: bulk-new %d notes", len(notes)))
}
