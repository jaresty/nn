package gitlocal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// CommitItem is a single unit of work queued for git commit.
type CommitItem struct {
	Op      string   `json:"op"`
	Message string   `json:"message"`
	Files   []string `json:"files"`
}

func queueDir(configDir string) string {
	return filepath.Join(configDir, "commit-queue")
}

func lockPath(configDir string) string {
	return filepath.Join(configDir, "commit-queue.lock")
}

// Enqueue atomically drops a CommitItem into the queue directory.
func Enqueue(configDir string, item CommitItem) error {
	dir := queueDir(configDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("queue: mkdir: %w", err)
	}
	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("queue: marshal: %w", err)
	}
	name := fmt.Sprintf("%d-%d.json", time.Now().UnixNano(), os.Getpid())
	tmp, err := os.CreateTemp(dir, "tmp-")
	if err != nil {
		return fmt.Errorf("queue: create temp: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return fmt.Errorf("queue: write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("queue: close temp: %w", err)
	}
	dest := filepath.Join(dir, name)
	if err := os.Rename(tmp.Name(), dest); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("queue: rename: %w", err)
	}
	return nil
}

// EnqueueAndDrain enqueues item then races to become the drainer.
// If another process wins the lock, this process exits after enqueuing.
func EnqueueAndDrain(configDir, repoDir string, item CommitItem) error {
	if err := Enqueue(configDir, item); err != nil {
		return err
	}
	return DrainQueue(configDir, repoDir)
}

// DrainQueue attempts to acquire the drain lock and commit all queued items.
// If another live process holds the lock, it returns nil (items will be drained
// by that process). Steals stale locks from dead processes.
func DrainQueue(configDir, repoDir string) error {
	lock := lockPath(configDir)
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		return fmt.Errorf("queue: mkdir lock dir: %w", err)
	}

	for {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			// Won the lock — write our pid and drain.
			fmt.Fprintf(f, "%d\n", os.Getpid())
			f.Close()
			defer os.Remove(lock)
			return drain(configDir, repoDir)
		}
		if !os.IsExist(err) {
			return fmt.Errorf("queue: open lock: %w", err)
		}
		// Lock exists — check if holder is alive.
		pid, readErr := readLockPid(lock)
		if readErr != nil || !pidAlive(pid) {
			// Stale lock — steal it.
			os.Remove(lock)
			continue
		}
		// Live holder — our item is queued and will be drained by them.
		return nil
	}
}

// drain commits all queued items in arrival order, looping until the queue is empty.
func drain(configDir, repoDir string) error {
	for {
		dir := queueDir(configDir)
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) || len(entries) == 0 {
			return nil
		}
		if err != nil {
			return fmt.Errorf("queue: read dir: %w", err)
		}

		// Filter and sort by filename (timestamp-pid prefix = arrival order).
		var names []string
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".json") {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)

		for _, name := range names {
			path := filepath.Join(dir, name)
			data, err := os.ReadFile(path)
			if os.IsNotExist(err) {
				continue // another drainer removed it (shouldn't happen, but safe)
			}
			if err != nil {
				return fmt.Errorf("queue: read item: %w", err)
			}
			var item CommitItem
			if err := json.Unmarshal(data, &item); err != nil {
				// Corrupt item — remove and skip.
				os.Remove(path)
				continue
			}
			if err := commitItem(repoDir, item); err != nil {
				return fmt.Errorf("queue: commit item: %w", err)
			}
			os.Remove(path)
		}
	}
}

func commitItem(repoDir string, item CommitItem) error {
	// Skip items whose files are outside the repo (e.g. stale test temp paths).
	for _, f := range item.Files {
		if !strings.HasPrefix(filepath.Clean(f), filepath.Clean(repoDir)) {
			return nil
		}
	}
	if item.Op == "delete" {
		// Stage removal; file is already gone from disk.
		rmArgs := append([]string{"rm", "--cached", "--ignore-unmatch"}, item.Files...)
		if out, err := gitCmd(repoDir, rmArgs...); err != nil {
			return fmt.Errorf("git rm: %w\n%s", err, out)
		}
		// Skip commit if nothing was staged (file was untracked).
		check := exec.Command("git", "diff", "--cached", "--quiet")
		check.Dir = repoDir
		if check.Run() == nil {
			return nil
		}
	} else {
		addArgs := append([]string{"add"}, item.Files...)
		if out, err := gitCmd(repoDir, addArgs...); err != nil {
			return fmt.Errorf("git add: %w\n%s", err, out)
		}
	}
	if out, err := gitCmd(repoDir, "commit", "-m", item.Message); err != nil {
		return fmt.Errorf("git commit: %w\n%s", err, out)
	}
	return nil
}

func gitCmd(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

func readLockPid(lock string) (int, error) {
	data, err := os.ReadFile(lock)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}
	return pid, nil
}

func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
