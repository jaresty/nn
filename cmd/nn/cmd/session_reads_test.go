package cmd

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestAccessLogIncludesPPID verifies that nn show <id> writes a PPID field to access.log.
func TestAccessLogIncludesPPID(t *testing.T) {
	_, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	out, err := execute("new", "--title", "ppid test note", "--type", "observation", "--no-edit")
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	id := strings.TrimPrefix(strings.TrimSpace(out), "created ")

	if _, err := execute("show", id); err != nil {
		t.Fatalf("show: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(cfgDir, "access.log"))
	if err != nil {
		t.Fatalf("read access.log: %v", err)
	}
	line := strings.TrimSpace(string(data))
	parts := strings.Fields(line)
	// format: <RFC3339> <PPID> show <id>
	if len(parts) < 4 {
		t.Fatalf("access.log line has %d fields, want 4: %q", len(parts), line)
	}
	ppid := parts[1]
	if _, err := strconv.Atoi(ppid); err != nil {
		t.Fatalf("access.log field[1] %q is not a PID integer", ppid)
	}
	if parts[2] != "show" {
		t.Fatalf("access.log field[2] = %q, want \"show\"", parts[2])
	}
}

// TestAccessLogGlobalSentinel verifies that nn show --global writes a --global sentinel to access.log.
func TestAccessLogGlobalSentinel(t *testing.T) {
	_, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	if _, err := execute("show", "--global"); err != nil {
		t.Fatalf("show --global: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(cfgDir, "access.log"))
	if err != nil {
		t.Fatalf("read access.log: %v", err)
	}
	if !strings.Contains(string(data), "--global") {
		t.Fatalf("access.log does not contain --global sentinel; got: %q", string(data))
	}
}

// TestLoadSessionReadsReturnsIDsAfterGlobal verifies that loadSessionReads returns IDs
// seen after the most recent --global entry for the current PPID.
func TestLoadSessionReadsReturnsIDsAfterGlobal(t *testing.T) {
	cfgDir := t.TempDir()
	ppid := os.Getppid()
	ppidStr := strconv.Itoa(ppid)

	// Write a --global sentinel, then a show entry.
	logPath := filepath.Join(cfgDir, "access.log")
	lines := []string{
		"2026-01-01T00:00:00Z " + ppidStr + " show --global",
		"2026-01-01T00:00:01Z " + ppidStr + " show abc123",
	}
	if err := os.WriteFile(logPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write access.log: %v", err)
	}

	reads := loadSessionReads(cfgDir)
	if !reads["abc123"] {
		t.Fatalf("loadSessionReads: want abc123 in set, got %v", reads)
	}
}

// TestListAnnotatesSessionReads verifies that nn list --search marks notes read this session with [read].
func TestListAnnotatesSessionReads(t *testing.T) {
	_, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	out, err := execute("new", "--title", "session annotation note", "--type", "observation", "--no-edit")
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	id := strings.TrimPrefix(strings.TrimSpace(out), "created ")

	// Establish --global sentinel then show the note to record it as read.
	if _, err := execute("show", "--global"); err != nil {
		t.Fatalf("show --global: %v", err)
	}
	if _, err := execute("show", id); err != nil {
		t.Fatalf("show: %v", err)
	}

	listOut, err := execute("list", "--search", "session annotation note")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(listOut, "[read]") {
		t.Fatalf("list output does not contain [read] marker; got:\n%s", listOut)
	}
}

// TestPruneWindow7Days verifies that entries older than 7 days are pruned (not 30).
func TestPruneWindow7Days(t *testing.T) {
	cfgDir := t.TempDir()
	logPath := filepath.Join(cfgDir, "access.log")

	// Write entry 8 days old (should be pruned) and 6 days old (should be kept).
	eightDaysAgo := time.Now().UTC().Add(-8 * 24 * time.Hour).Format(time.RFC3339)
	sixDaysAgo := time.Now().UTC().Add(-6 * 24 * time.Hour).Format(time.RFC3339)
	content := eightDaysAgo + " 99999 show eight-days-old\n" +
		sixDaysAgo + " 99999 show six-days-old\n"
	// Make the file large enough to trigger pruning.
	padding := strings.Repeat("2026-01-01T00:00:00Z 99999 show pad-id\n", 1500)
	if err := os.WriteFile(logPath, []byte(padding+content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	appendAccessLog(cfgDir, "new-id")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(data), "eight-days-old") {
		t.Fatalf("access.log still contains 8-day-old entry; got:\n%s", string(data))
	}
	if !strings.Contains(string(data), "six-days-old") {
		t.Fatalf("access.log dropped 6-day-old entry; got:\n%s", string(data))
	}
}

// TestPruneSkipsWhenUnderSizeThreshold verifies pruning is skipped when file is under 50KB.
func TestPruneSkipsWhenUnderSizeThreshold(t *testing.T) {
	cfgDir := t.TempDir()
	logPath := filepath.Join(cfgDir, "access.log")

	// Write an old entry but keep file small (under 50KB).
	old := "2020-01-01T00:00:00Z 99999 show old-small-file\n"
	if err := os.WriteFile(logPath, []byte(old), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	appendAccessLog(cfgDir, "new-id")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	// Old entry must survive because file was under 50KB threshold.
	if !strings.Contains(string(data), "old-small-file") {
		t.Fatalf("access.log pruned entry despite file being under 50KB; got:\n%s", string(data))
	}
}

// TestPruneSkipsWhenRecentlyPruned verifies pruning is skipped if last prune was under 1 hour ago.
func TestPruneSkipsWhenRecentlyPruned(t *testing.T) {
	cfgDir := t.TempDir()
	logPath := filepath.Join(cfgDir, "access.log")
	pruneMarkerPath := filepath.Join(cfgDir, "access.log.pruned")

	// Write a large file with an old entry.
	old := "2020-01-01T00:00:00Z 99999 show old-recently-pruned\n"
	padding := strings.Repeat("2026-01-01T00:00:00Z 99999 show pad-id\n", 1500)
	if err := os.WriteFile(logPath, []byte(padding+old), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	// Write a prune marker timestamped 23 hours ago (within 24-hour buffer, but outside 1-hour buffer).
	recentPrune := time.Now().UTC().Add(-23 * time.Hour).Format(time.RFC3339)
	if err := os.WriteFile(pruneMarkerPath, []byte(recentPrune), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	appendAccessLog(cfgDir, "new-id")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	// Old entry must survive because last prune was recent.
	if !strings.Contains(string(data), "old-recently-pruned") {
		t.Fatalf("access.log pruned entry despite recent prune marker; got:\n%s", string(data))
	}
}

// TestAccessLogPrunesOldEntries verifies that entries older than 7 days are removed on write when conditions are met.
func TestAccessLogPrunesOldEntries(t *testing.T) {
	cfgDir := t.TempDir()
	logPath := filepath.Join(cfgDir, "access.log")

	// Write one old entry (40 days ago) with enough padding to exceed 50KB threshold.
	old := "2020-01-01T00:00:00Z 99999 show old-id"
	padding := strings.Repeat("2026-01-01T00:00:00Z 99999 show pad-id\n", 1500)
	if err := os.WriteFile(logPath, []byte(padding+old+"\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	appendAccessLog(cfgDir, "new-id")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(data), "old-id") {
		t.Fatalf("access.log still contains old entry after prune; got:\n%s", string(data))
	}
	if !strings.Contains(string(data), "new-id") {
		t.Fatalf("access.log does not contain new entry; got:\n%s", string(data))
	}
}
