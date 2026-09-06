package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// twoSessionDir builds a dir with two sessions of different mtimes:
// "old.jsonl" (older) and an sdk-cli session "new.jsonl" (newer) with subagents.
func twoSessionDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	old := filepath.Join(dir, "old.jsonl")
	writeTranscriptFile(t, old, `{"type":"assistant","uuid":"o1","message":{"role":"assistant","content":[],"usage":{"input_tokens":5,"output_tokens":5}}}`+"\n")

	// sdk-cli session with a small tree.
	writeSDKCLIFixture(t, dir) // creates sess.jsonl + subagents

	// set mtimes: old older than sess.
	oldTime := time.Now().Add(-2 * time.Hour)
	newTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(old, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(dir, "sess.jsonl"), newTime, newTime); err != nil {
		t.Fatal(err)
	}
	return dir
}

// Assertion [14]: ls lists sessions most-recent-first with schema, agent count, cost.
func TestTranscriptLsListsRecentFirst(t *testing.T) {
	dir := twoSessionDir(t)
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "ls", dir)
	if err != nil {
		t.Fatalf("nn transcript ls: %v", err)
	}
	// newer sdk-cli session ("sess") should appear before the older one ("old").
	sessIdx := strings.Index(out, "sess")
	oldIdx := strings.Index(out, "old")
	if sessIdx == -1 || oldIdx == -1 {
		t.Fatalf("expected both sessions listed:\n%s", out)
	}
	if sessIdx > oldIdx {
		t.Errorf("expected newer session (sess) listed before older (old):\n%s", out)
	}
	if !strings.Contains(out, "sdk-cli") {
		t.Errorf("expected schema label in ls output:\n%s", out)
	}
}

// Assertion [15]: each row includes a compact inline mini-tree preview.
func TestTranscriptLsMiniTreePreview(t *testing.T) {
	dir := twoSessionDir(t)
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "ls", dir)
	if err != nil {
		t.Fatalf("nn transcript ls: %v", err)
	}
	// the sdk-cli fixture has ROOT + aaa + bbb; a mini-tree preview names ROOT.
	if !strings.Contains(out, "ROOT") {
		t.Errorf("expected a mini-tree preview containing ROOT:\n%s", out)
	}
}

// Assertion [16]: --limit caps rows; --before pages older.
func TestTranscriptLsLimitAndBefore(t *testing.T) {
	dir := twoSessionDir(t)
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "ls", dir, "--limit", "1")
	if err != nil {
		t.Fatalf("nn transcript ls --limit: %v", err)
	}
	// only the newest (sess) should appear, not old.
	if strings.Contains(out, "old.jsonl") || strings.Contains(out, "old ") {
		t.Errorf("--limit 1 should exclude the older session:\n%s", out)
	}

	// --before 90m ago should exclude the newer (1h-old) session, include old (2h).
	before := time.Now().Add(-90 * time.Minute).UTC().Format(time.RFC3339)
	out2, err := execute("transcript", "ls", dir, "--before", before)
	if err != nil {
		t.Fatalf("nn transcript ls --before: %v", err)
	}
	if !strings.Contains(out2, "old") {
		t.Errorf("--before should include the older session:\n%s", out2)
	}
	if strings.Contains(out2, "sess.jsonl") {
		t.Errorf("--before 90m should exclude the 1h-old session:\n%s", out2)
	}
}

// Assertion [17]: --json emits structured session rows.
func TestTranscriptLsJSON(t *testing.T) {
	dir := twoSessionDir(t)
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "ls", dir, "--json")
	if err != nil {
		t.Fatalf("nn transcript ls --json: %v", err)
	}
	var rows []struct {
		Session     string `json:"session"`
		Path        string `json:"path"`
		Modified    string `json:"modified"`
		Schema      string `json:"schema"`
		AgentCount  int    `json:"agent_count"`
		TotalCost   int    `json:"total_cost"`
		TreePreview string `json:"tree_preview"`
	}
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("parse ls json: %v\n%s", err, out)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 session rows, got %d:\n%s", len(rows), out)
	}
	// newest first
	if rows[0].Schema != "sdk-cli" {
		t.Errorf("expected newest row to be sdk-cli, got %q", rows[0].Schema)
	}
	if rows[0].AgentCount != 3 { // ROOT + aaa + bbb
		t.Errorf("expected sdk-cli agent_count 3, got %d", rows[0].AgentCount)
	}
	if rows[0].TreePreview == "" {
		t.Errorf("expected non-empty tree_preview in json row")
	}
}
