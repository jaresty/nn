package feedback

import (
	"os"
	"path/filepath"
	"time"
)

// FeedbackRoot returns the directory holding all feedback session directories:
// <configdir>/feedback/.
func FeedbackRoot() string {
	return filepath.Join(configDir(), "feedback")
}

// CleanupSessions removes feedback session directories under root whose most
// recent file was modified longer ago than retention. It is best-effort: a
// missing root is not an error, and a failure to remove one directory does not
// stop the sweep or fail the caller. The session directory is ephemeral scratch
// — the durable artifact is whatever the agent chose to persist from the result
// — so aged sessions are safe to reclaim.
func CleanupSessions(root string, retention time.Duration) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff := time.Now().Add(-retention)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		if newestMtime(dir).After(cutoff) {
			continue // still within retention window
		}
		_ = os.RemoveAll(dir) // best-effort
	}
	return nil
}

// newestMtime returns the most recent modification time among the files in dir
// (and the dir itself). Using the newest file's mtime, rather than the
// directory's own, is reliable across platforms where a dir mtime does not
// track changes to files within it.
func newestMtime(dir string) time.Time {
	newest := time.Time{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Can't inspect files; fall back to the directory's own mtime.
		if info, serr := os.Stat(dir); serr == nil {
			return info.ModTime()
		}
		return newest
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
	}
	return newest
}
