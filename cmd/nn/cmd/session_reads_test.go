package cmd

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/backend/gitlocal"
)

// TestResolveInstructionSuppressedWhenAllRead verifies that nn grep suppresses the resolve
// instruction when all related notes are already read this session.
func TestResolveInstructionSuppressedWhenAllRead(t *testing.T) {
	_, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	setupSessionRead(t, execute, "suppressmarker unique resolve suppression test")

	f := filepath.Join(t.TempDir(), "suppress.go")
	if err := os.WriteFile(f, []byte("// suppressmarker unique resolve suppression test\npackage main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "suppressmarker", f)
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	if strings.Contains(out, "Resolve each") {
		t.Fatalf("resolve instruction present despite all notes being read; got:\n%s", out)
	}
}

// TestResolveInstructionPresentWhenUnread verifies that nn grep emits the resolve instruction
// when at least one related note has not been read this session.
func TestResolveInstructionPresentWhenUnread(t *testing.T) {
	_, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	// Create note but do NOT show it — it remains unread.
	out, err := execute("new", "--title", "unreadmarker unique unread resolve test", "--type", "observation", "--no-edit")
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	_ = strings.TrimPrefix(strings.TrimSpace(out), "created ")

	// Fire --global so a session exists, but don't show the note.
	if _, err := execute("show", "--global"); err != nil {
		t.Fatalf("show --global: %v", err)
	}

	f := filepath.Join(t.TempDir(), "unread.go")
	if err := os.WriteFile(f, []byte("// unreadmarker unique unread resolve test\npackage main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	grepOut, err := execute("grep", "unreadmarker", f)
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	if !strings.Contains(grepOut, "Resolve each unread") {
		t.Fatalf("resolve instruction absent for unread note; got:\n%s", grepOut)
	}
}

// TestResolveInstructionTextNamesRead verifies the resolve instruction explicitly states
// that [read] notes do not require action.
func TestResolveInstructionTextNamesRead(t *testing.T) {
	_, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	out, err := execute("new", "--title", "readtextmarker unique instruction text test", "--type", "observation", "--no-edit")
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	_ = strings.TrimPrefix(strings.TrimSpace(out), "created ")

	if _, err := execute("show", "--global"); err != nil {
		t.Fatalf("show --global: %v", err)
	}

	f := filepath.Join(t.TempDir(), "readtext.go")
	if err := os.WriteFile(f, []byte("// readtextmarker unique instruction text test\npackage main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	grepOut, err := execute("grep", "readtextmarker", f)
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	if !strings.Contains(grepOut, "Notes marked [read] have already") {
		t.Fatalf("resolve instruction does not mention [read] notes; got:\n%s", grepOut)
	}
}

// TestListResolveInstructionForUnread verifies nn list --search emits resolve instruction
// for unread BM25 results.
func TestListResolveInstructionForUnread(t *testing.T) {
	_, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	out, err := execute("new", "--title", "listresolvemarker unique list resolve test", "--type", "observation", "--no-edit")
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	_ = strings.TrimPrefix(strings.TrimSpace(out), "created ")

	if _, err := execute("show", "--global"); err != nil {
		t.Fatalf("show --global: %v", err)
	}

	listOut, err := execute("list", "--search", "listresolvemarker unique list resolve test")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(listOut, "Resolve each unread") {
		t.Fatalf("list --search resolve instruction absent for unread note; got:\n%s", listOut)
	}
}

// TestReadResolveInstructionSuppressedWhenAllRead verifies nn read suppresses resolve when all read.
func TestReadResolveInstructionSuppressedWhenAllRead(t *testing.T) {
	_, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)
	setupSessionRead(t, execute, "readsuppressmarker unique read suppress test")
	f := filepath.Join(t.TempDir(), "rs.go")
	if err := os.WriteFile(f, []byte("// readsuppressmarker unique read suppress test\npackage main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := execute("read", f)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(out, "Resolve each") {
		t.Fatalf("resolve instruction present despite all notes being read; got:\n%s", out)
	}
}

// TestTeeResolveInstructionSuppressedWhenAllRead verifies nn tee suppresses resolve when all read.
func TestTeeResolveInstructionSuppressedWhenAllRead(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)
	id := setupSessionRead(t, execute, "teesuppressmarker unique tee suppress test")
	gl, err := gitlocal.New(nbDir)
	if err != nil {
		t.Fatalf("backend: %v", err)
	}
	state := &rootState{notebookDir: nbDir, backend: gl}
	sessionReads := map[string]bool{id: true}
	var stdout, stderrBuf strings.Builder
	if err := runTee(strings.NewReader("teesuppressmarker unique tee suppress test"), &stdout, &stderrBuf, state, sessionReads); err != nil {
		t.Fatalf("runTee: %v", err)
	}
	if strings.Contains(stderrBuf.String(), "Resolve each") {
		t.Fatalf("resolve instruction present despite all notes being read; got:\n%s", stderrBuf.String())
	}
}

// TestAstResolveInstructionSuppressedWhenAllRead verifies nn ast suppresses resolve when all read.
func TestAstResolveInstructionSuppressedWhenAllRead(t *testing.T) {
	_, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)
	setupSessionRead(t, execute, "AstSuppressMarker unique ast suppress test")
	f := filepath.Join(t.TempDir(), "astsup.go")
	if err := os.WriteFile(f, []byte("package main\n\nfunc AstSuppressMarker() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := execute("ast", f)
	if err != nil {
		t.Fatalf("ast: %v", err)
	}
	if strings.Contains(out, "Resolve each") {
		t.Fatalf("resolve instruction present despite all notes being read; got:\n%s", out)
	}
}

// TestTraceResolveInstructionSuppressedWhenAllRead verifies nn trace suppresses resolve when all read.
func TestTraceResolveInstructionSuppressedWhenAllRead(t *testing.T) {
	_, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)
	setupSessionRead(t, execute, "TraceSuppressMarker unique trace suppress test")
	dir := t.TempDir()
	f := filepath.Join(dir, "main.go")
	if err := os.WriteFile(f, []byte("package main\n\nfunc TraceSuppressMarker() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := execute("trace", dir, "--symbol", "TraceSuppressMarker")
	if err != nil {
		t.Fatalf("trace: %v", err)
	}
	if strings.Contains(out, "Resolve each") {
		t.Fatalf("resolve instruction present despite all notes being read; got:\n%s", out)
	}
}

// setupSessionRead creates a note, fires nn show --global, then nn show <id> to register the
// note as read this session. Returns the note ID.
func setupSessionRead(t *testing.T, execute func(...string) (string, error), title string) string {
	t.Helper()
	out, err := execute("new", "--title", title, "--type", "observation", "--no-edit")
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	id := strings.TrimPrefix(strings.TrimSpace(out), "created ")
	if _, err := execute("show", "--global"); err != nil {
		t.Fatalf("show --global: %v", err)
	}
	if _, err := execute("show", id); err != nil {
		t.Fatalf("show %s: %v", id, err)
	}
	return id
}

// TestGrepShowsReadMarker verifies nn grep annotates related notes with [read] when session-read.
func TestGrepShowsReadMarker(t *testing.T) {
	_, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	// Create a note with distinctive content, register it as read.
	setupSessionRead(t, execute, "grepmarker unique authentication flow")

	// Create a file whose content overlaps the note title to trigger BM25 match.
	f := filepath.Join(t.TempDir(), "auth.go")
	if err := os.WriteFile(f, []byte("// grepmarker unique authentication flow\npackage main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("grep", "grepmarker", f)
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	if !strings.Contains(out, "[read]") {
		t.Fatalf("nn grep output does not contain [read] marker; got:\n%s", out)
	}
}

// TestReadShowsReadMarker verifies nn read annotates related notes with [read] when session-read.
func TestReadShowsReadMarker(t *testing.T) {
	_, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	setupSessionRead(t, execute, "readmarker unique token validation")

	f := filepath.Join(t.TempDir(), "token.go")
	if err := os.WriteFile(f, []byte("// readmarker unique token validation\npackage main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("read", f)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(out, "[read]") {
		t.Fatalf("nn read output does not contain [read] marker; got:\n%s", out)
	}
}

// TestTeeShowsReadMarker verifies nn tee annotates related notes with [read] when session-read.
func TestTeeShowsReadMarker(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	id := setupSessionRead(t, execute, "teemarker unique session pipe flow")

	// Build rootState directly so runTee can query notes.
	gl, err := gitlocal.New(nbDir)
	if err != nil {
		t.Fatalf("backend: %v", err)
	}
	state := &rootState{notebookDir: nbDir, backend: gl}

	sessionReads := map[string]bool{id: true}

	var stdout, stderrBuf strings.Builder
	input := strings.NewReader("teemarker unique session pipe flow")
	if err := runTee(input, &stdout, &stderrBuf, state, sessionReads); err != nil {
		t.Fatalf("runTee: %v", err)
	}
	if !strings.Contains(stderrBuf.String(), "[read]") {
		t.Fatalf("nn tee stderr does not contain [read] marker; got:\n%s", stderrBuf.String())
	}
}

// TestAstShowsReadMarker verifies nn ast annotates related notes with [read] when session-read.
func TestAstShowsReadMarker(t *testing.T) {
	_, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	setupSessionRead(t, execute, "AstmarkerAnalyze unique symbol analysis")

	f := filepath.Join(t.TempDir(), "sym.go")
	// Function name matches note title for BM25 overlap via ast symbol query.
	if err := os.WriteFile(f, []byte("package main\n\nfunc AstmarkerAnalyze() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("ast", f)
	if err != nil {
		t.Fatalf("ast: %v", err)
	}
	if !strings.Contains(out, "[read]") {
		t.Fatalf("nn ast output does not contain [read] marker; got:\n%s", out)
	}
}

// TestTraceShowsReadMarker verifies nn trace annotates related notes with [read] when session-read.
func TestTraceShowsReadMarker(t *testing.T) {
	_, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	// Note title matches symbol name so BM25 query on symbol body/name produces a hit.
	setupSessionRead(t, execute, "TracemarkerEntry unique call graph traversal")

	dir := t.TempDir()
	f := filepath.Join(dir, "main.go")
	if err := os.WriteFile(f, []byte("package main\n\nfunc TracemarkerEntry() { TracemarkerInner() }\nfunc TracemarkerInner() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("trace", dir, "--symbol", "TracemarkerEntry")
	if err != nil {
		t.Fatalf("trace: %v", err)
	}
	if !strings.Contains(out, "[read]") {
		t.Fatalf("nn trace output does not contain [read] marker; got:\n%s", out)
	}
}

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
