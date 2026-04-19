// Package gitlocal implements the Backend interface using the local filesystem
// with a Git repository for history. Each write operation produces one commit.
package gitlocal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jaresty/nn/internal/backend"
	"github.com/jaresty/nn/internal/note"
)

// Backend stores notes as Markdown files in a Git-backed directory.
type Backend struct {
	dir string
}

// New returns a Backend rooted at dir, which must already be a Git repository.
func New(dir string) (*Backend, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("gitlocal.New: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("gitlocal.New: %q is not a directory", dir)
	}
	return &Backend{dir: dir}, nil
}

// Write serialises n to a Markdown file and commits it to Git.
func (b *Backend) Write(n *note.Note) error {
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
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("gitlocal.Write: %w", err)
	}
	msg := fmt.Sprintf("note: create %s — %s", n.ID, n.Title)
	return b.commit(path, msg)
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
	msg := fmt.Sprintf("note: delete %s — %s", n.ID, n.Title)
	return b.commitDelete(path, msg)
}

// List returns all notes in the notebook directory.
func (b *Backend) List() ([]*note.Note, error) {
	entries, err := os.ReadDir(b.dir)
	if err != nil {
		return nil, fmt.Errorf("gitlocal.List: %w", err)
	}
	var notes []*note.Note
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(b.dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("gitlocal.List: read %s: %w", e.Name(), err)
		}
		n, err := note.Parse(data)
		if err != nil {
			// Skip unparseable files (e.g., README.md without frontmatter).
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

// commit stages the file at path and creates a commit with msg.
func (b *Backend) commit(path, msg string) error {
	if err := b.git("add", path); err != nil {
		return err
	}
	return b.git("commit", "-m", msg)
}

// commitDelete stages the deleted file and creates a commit with msg.
// If the file was not tracked by git (e.g. written outside the backend in tests),
// the commit is skipped gracefully.
func (b *Backend) commitDelete(path, msg string) error {
	_ = b.git("rm", "--cached", "--ignore-unmatch", path) // best-effort; ignore error for untracked
	// Check whether anything is staged before committing.
	check := exec.Command("git", "diff", "--cached", "--quiet")
	check.Dir = b.dir
	if check.Run() == nil {
		return nil // nothing staged — file was not tracked, deletion is still done on disk
	}
	return b.git("commit", "-m", msg)
}

// git runs a git subcommand in the backend directory.
func (b *Backend) git(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = b.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return nil
}

// AddLink adds an annotated link from fromID to toID and commits.
func (b *Backend) AddLink(fromID, toID, annotation, linkType, linkStatus string) error {
	n, err := b.Read(fromID)
	if err != nil {
		return fmt.Errorf("gitlocal.AddLink: %w", err)
	}
	for _, lnk := range n.Links {
		if lnk.TargetID == toID {
			return fmt.Errorf("gitlocal.AddLink: link %s→%s already exists", fromID, toID)
		}
	}
	n.Links = append(n.Links, note.Link{TargetID: toID, Annotation: annotation, Type: linkType, Status: linkStatus})
	data, err := n.Marshal()
	if err != nil {
		return fmt.Errorf("gitlocal.AddLink: marshal: %w", err)
	}
	path := filepath.Join(b.dir, n.Filename())
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("gitlocal.AddLink: write: %w", err)
	}
	msg := fmt.Sprintf("note: link %s → %s", fromID, toID)
	return b.commit(path, msg)
}

// AddLinks adds multiple annotated links from fromID in a single git commit.
func (b *Backend) AddLinks(fromID string, targets []backend.LinkTarget) error {
	n, err := b.Read(fromID)
	if err != nil {
		return fmt.Errorf("gitlocal.AddLinks: %w", err)
	}
	existing := make(map[string]bool, len(n.Links))
	for _, lnk := range n.Links {
		existing[lnk.TargetID] = true
	}
	for _, t := range targets {
		if existing[t.ToID] {
			return fmt.Errorf("gitlocal.AddLinks: link %s→%s already exists", fromID, t.ToID)
		}
		n.Links = append(n.Links, note.Link{TargetID: t.ToID, Annotation: t.Annotation, Type: t.Type, Status: t.Status})
		existing[t.ToID] = true
	}
	data, err := n.Marshal()
	if err != nil {
		return fmt.Errorf("gitlocal.AddLinks: marshal: %w", err)
	}
	path := filepath.Join(b.dir, n.Filename())
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("gitlocal.AddLinks: write: %w", err)
	}
	msg := fmt.Sprintf("note: bulk-link %s → %d notes", fromID, len(targets))
	return b.commit(path, msg)
}

// RemoveLink removes the link from fromID to toID and commits.
func (b *Backend) RemoveLink(fromID, toID string) error {
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
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("gitlocal.RemoveLink: write: %w", err)
	}
	msg := fmt.Sprintf("note: unlink %s → %s", fromID, toID)
	return b.commit(path, msg)
}

// BulkUpdateLinks applies multiple link updates to fromID in a single git commit.
func (b *Backend) BulkUpdateLinks(fromID string, updates []backend.LinkUpdate) error {
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
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("gitlocal.BulkUpdateLinks: write: %w", err)
	}
	msg := fmt.Sprintf("note: bulk-update-link %s (%d links)", fromID, len(updates))
	return b.commit(path, msg)
}

// UpdateLink modifies the annotation, type, and/or status of an existing link without removing it.
// nil pointer arguments mean "leave unchanged".
func (b *Backend) UpdateLink(fromID, toID string, annotation, linkType, linkStatus *string) error {
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
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("gitlocal.UpdateLink: write: %w", err)
	}
	msg := fmt.Sprintf("note: update-link %s → %s", fromID, toID)
	return b.commit(path, msg)
}

// Update writes the modified note and commits with an "update" message.
func (b *Backend) Update(n *note.Note) error {
	data, err := n.Marshal()
	if err != nil {
		return fmt.Errorf("gitlocal.Update: %w", err)
	}
	path := filepath.Join(b.dir, n.Filename())
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("gitlocal.Update: %w", err)
	}
	msg := fmt.Sprintf("note: update %s — %s", n.ID, n.Title)
	return b.commit(path, msg)
}

// Promote updates the status of the note with the given id and commits.
func (b *Backend) Promote(id string, to note.Status) error {
	n, err := b.Read(id)
	if err != nil {
		return fmt.Errorf("gitlocal.Promote: %w", err)
	}
	n.Status = to
	data, err := n.Marshal()
	if err != nil {
		return fmt.Errorf("gitlocal.Promote: marshal: %w", err)
	}
	path := filepath.Join(b.dir, n.Filename())
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("gitlocal.Promote: write: %w", err)
	}
	msg := fmt.Sprintf("note: promote %s to %s", id, string(to))
	return b.commit(path, msg)
}

// BulkWrite writes all notes and commits in a single commit.
func (b *Backend) BulkWrite(notes []*note.Note) error {
	if len(notes) == 0 {
		return nil
	}
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
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("gitlocal.BulkWrite: write %s: %w", n.ID, err)
		}
		if err := b.git("add", path); err != nil {
			return fmt.Errorf("gitlocal.BulkWrite: stage %s: %w", n.ID, err)
		}
	}
	return b.git("commit", "-m", fmt.Sprintf("note: bulk-new %d notes", len(notes)))
}
