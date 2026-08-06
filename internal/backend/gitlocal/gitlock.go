package gitlocal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type processGitLockEntry struct {
	mu   sync.Mutex
	refs int
}

var processGitLockRegistry = struct {
	sync.Mutex
	entries map[string]*processGitLockEntry
}{entries: make(map[string]*processGitLockEntry)}

func canonicalGitLockDir(configDir string) string {
	absolute, err := filepath.Abs(configDir)
	if err != nil {
		absolute = filepath.Clean(configDir)
	}
	current := absolute
	var unresolved []string
	for {
		if resolved, err := filepath.EvalSymlinks(current); err == nil {
			for i := len(unresolved) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, unresolved[i])
			}
			return filepath.Clean(resolved)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return filepath.Clean(absolute)
		}
		unresolved = append(unresolved, filepath.Base(current))
		current = parent
	}
}

func gitLockPath(configDir string) string {
	return filepath.Join(canonicalGitLockDir(configDir), "git-commit.lock")
}

// acquireGitLock spins until it wins an O_EXCL lock file, stealing stale locks
// from dead processes. Holds the lock for the duration of the git operation.
// AcquireGitLock is the exported form for use in tests.
func AcquireGitLock(configDir string) error { return acquireGitLock(configDir) }

// ReleaseGitLock is the exported form for use in tests.
func ReleaseGitLock(configDir string) { releaseGitLock(configDir) }

func retainProcessGitLock(configDir string) *processGitLockEntry {
	lockPath := filepath.Clean(gitLockPath(configDir))
	processGitLockRegistry.Lock()
	defer processGitLockRegistry.Unlock()
	entry := processGitLockRegistry.entries[lockPath]
	if entry == nil {
		entry = &processGitLockEntry{}
		processGitLockRegistry.entries[lockPath] = entry
	}
	entry.refs++
	return entry
}

func releaseProcessGitLock(configDir string, entry *processGitLockEntry) {
	entry.mu.Unlock()
	lockPath := filepath.Clean(gitLockPath(configDir))
	processGitLockRegistry.Lock()
	defer processGitLockRegistry.Unlock()
	entry.refs--
	if entry.refs == 0 {
		delete(processGitLockRegistry.entries, lockPath)
	}
}

func acquireGitLock(configDir string) error {
	processLock := retainProcessGitLock(configDir)
	processLock.mu.Lock()
	acquired := false
	defer func() {
		if !acquired {
			releaseProcessGitLock(configDir, processLock)
		}
	}()
	lock := gitLockPath(configDir)
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		return fmt.Errorf("gitlock: mkdir: %w", err)
	}
	for {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			fmt.Fprintf(f, "%d\n", os.Getpid())
			if err := f.Close(); err != nil {
				os.Remove(lock)
				return fmt.Errorf("gitlock: write pid: %w", err)
			}
			acquired = true
			return nil
		}
		if !os.IsExist(err) {
			return fmt.Errorf("gitlock: open: %w", err)
		}
		// Lock exists — check if holder is alive.
		pid, err := readGitLockPid(lock)
		if err != nil {
			// Empty or unreadable — writer is between O_EXCL create and PID write.
			// Wait rather than stealing to avoid a false-stale race.
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if !gitLockPidAlive(pid) {
			os.Remove(lock) // stale — steal it
			continue
		}
		if pid == os.Getpid() {
			// A stale same-process lock cannot belong to another active backend:
			// the lock-path mutex serializes all matching acquisitions in this process.
			acquired = true
			return nil
		}
		// Live holder — wait and retry.
		time.Sleep(10 * time.Millisecond)
	}
}

func releaseGitLock(configDir string) {
	os.Remove(gitLockPath(configDir))
	lockPath := filepath.Clean(gitLockPath(configDir))
	processGitLockRegistry.Lock()
	entry := processGitLockRegistry.entries[lockPath]
	processGitLockRegistry.Unlock()
	if entry != nil {
		releaseProcessGitLock(configDir, entry)
	}
}

func readGitLockPid(lock string) (int, error) {
	data, err := os.ReadFile(lock)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func gitLockPidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// gitCmdIn runs a git subcommand in dir and returns combined output.
func gitCmdIn(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

// nothingStaged returns true when git reports no staged changes.
func nothingStaged(dir string) bool {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = dir
	return cmd.Run() == nil
}
