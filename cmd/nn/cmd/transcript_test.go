package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile is a small helper for building transcript fixtures.
func writeTranscriptFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// Assertion [1]: a dir with a subagents/agent-*.jsonl + .meta.json is classified sdk-cli.
func TestTranscriptScanClassifiesSDKCLI(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "sess1.jsonl")
	writeTranscriptFile(t, session, `{"type":"assistant","uuid":"u1","parentUuid":"u0"}`+"\n")
	sub := filepath.Join(dir, "sess1", "subagents", "agent-aaa.jsonl")
	writeTranscriptFile(t, sub, `{"type":"assistant","uuid":"s1","sourceToolAssistantUUID":"x"}`+"\n")
	meta := filepath.Join(dir, "sess1", "subagents", "agent-aaa.meta.json")
	writeTranscriptFile(t, meta, `{"agentType":"general-purpose","toolUseId":"toolu_1"}`)

	_, execute := setupNotebook(t)
	out, err := execute("transcript", "scan", dir)
	if err != nil {
		t.Fatalf("nn transcript scan: %v", err)
	}
	if !strings.Contains(out, "sdk-cli") {
		t.Errorf("expected sdk-cli classification in output:\n%s", out)
	}
}

// Assertion [2]: a dir with a pi transcript (custom subagents:record + Agent toolCall) is classified pi.
func TestTranscriptScanClassifiesPi(t *testing.T) {
	dir := t.TempDir()
	pi := filepath.Join(dir, "pi-sess.jsonl")
	writeTranscriptFile(t, pi,
		`{"type":"session","version":3,"id":"01a","cwd":"/x"}`+"\n"+
			`{"type":"message","id":"m1","parentId":"a0","message":{"role":"assistant","content":[{"type":"toolCall","id":"call_1","name":"Agent","arguments":{"subagent_type":"general-purpose"}}]}}`+"\n"+
			`{"type":"custom","customType":"subagents:record","id":"c1","parentId":"m1","data":{"id":"d1","status":"completed","result":"ok"}}`+"\n")

	_, execute := setupNotebook(t)
	out, err := execute("transcript", "scan", dir)
	if err != nil {
		t.Fatalf("nn transcript scan: %v", err)
	}
	if !strings.Contains(out, "pi") {
		t.Errorf("expected pi classification in output:\n%s", out)
	}
}

// Assertion [3]: a dir with an interactive Claude Code transcript (parentUuid + inline Task, no subagents/) is claude-code.
func TestTranscriptScanClassifiesClaudeCode(t *testing.T) {
	dir := t.TempDir()
	cc := filepath.Join(dir, "cc-sess.jsonl")
	writeTranscriptFile(t, cc,
		`{"type":"user","uuid":"u0","message":{"role":"user","content":"hi"}}`+"\n"+
			`{"type":"assistant","uuid":"u1","parentUuid":"u0","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Task","input":{"subagent_type":"general-purpose"}}]}}`+"\n")

	_, execute := setupNotebook(t)
	out, err := execute("transcript", "scan", dir)
	if err != nil {
		t.Fatalf("nn transcript scan: %v", err)
	}
	if !strings.Contains(out, "claude-code") {
		t.Errorf("expected claude-code classification in output:\n%s", out)
	}
}

// Assertion [4]: an unrecognizable .jsonl is reported under unknown.
func TestTranscriptScanReportsUnknown(t *testing.T) {
	dir := t.TempDir()
	writeTranscriptFile(t, filepath.Join(dir, "weird.jsonl"), `{"foo":"bar","baz":1}`+"\n")

	_, execute := setupNotebook(t)
	out, err := execute("transcript", "scan", dir)
	if err != nil {
		t.Fatalf("nn transcript scan: %v", err)
	}
	if !strings.Contains(out, "unknown") {
		t.Errorf("expected unknown classification in output:\n%s", out)
	}
}

// Assertion [5]: scan reports a count per schema.
func TestTranscriptScanReportsCounts(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a", "b"} {
		writeTranscriptFile(t, filepath.Join(dir, n+".jsonl"), `{"foo":"bar"}`+"\n")
	}

	_, execute := setupNotebook(t)
	out, err := execute("transcript", "scan", dir)
	if err != nil {
		t.Fatalf("nn transcript scan: %v", err)
	}
	// two unknown transcripts → count of 2 for unknown
	if !strings.Contains(out, "unknown") || !strings.Contains(out, "2") {
		t.Errorf("expected unknown count of 2 in output:\n%s", out)
	}
}

// Assertion [6]: doctor reports duckdb availability and that it is escape-hatch-only.
func TestTranscriptDoctorReportsDuckDB(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("transcript", "doctor")
	if err != nil {
		t.Fatalf("nn transcript doctor: %v", err)
	}
	if !strings.Contains(strings.ToLower(out), "duckdb") {
		t.Errorf("expected duckdb mention in doctor output:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(out), "escape") {
		t.Errorf("expected escape-hatch note in doctor output:\n%s", out)
	}
}

// Assertion [7]: bare `nn transcript` (no subcommand) prints curated job-oriented help —
// names ls as the start, groups by job, and points at the patterns skill for cross-session work.
func TestTranscriptBareCommandPrintsJobHelp(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("transcript")
	if err != nil {
		t.Fatalf("nn transcript (bare): %v", err)
	}
	low := strings.ToLower(out)
	// names ls as the start
	if !strings.Contains(low, "ls") || !strings.Contains(low, "start here") {
		t.Errorf("expected bare help to name ls under a 'Start here' heading:\n%s", out)
	}
	// job-grouping heading (navigate one session)
	if !strings.Contains(low, "navigate one session") {
		t.Errorf("expected a job-grouping heading in bare help:\n%s", out)
	}
	// patterns tip pointing at the skill
	if !strings.Contains(low, "pattern") || !strings.Contains(low, "nn-transcript") {
		t.Errorf("expected a cross-session patterns tip naming the nn-transcript skill:\n%s", out)
	}
}
