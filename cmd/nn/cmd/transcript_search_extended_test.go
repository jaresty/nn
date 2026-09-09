package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranscriptSearchRegex(t *testing.T) {
	_, run := setupNotebook(t)
	p := filepath.Join(t.TempDir(), "a.jsonl")
	writeTranscriptFile(t, p, `{"type":"session","id":"s"}`+"\n"+`{"type":"message","id":"e","message":{"role":"assistant","content":"Trace_v2 3896"}}`+"\n"+`{"type":"message","id":"raw","message":{"role":"toolResult","content":"hidden_123"}}`+"\n")
	for _, tc := range []struct {
		q     string
		flags []string
		n     int
	}{
		{"trace_v2", nil, 1}, {"Trace_v[0-9]", nil, 0},
		{"^Trace_v[0-9] 389[0-9]$", []string{"--regex"}, 1},
		{"trace_v[0-9]", []string{"--regex"}, 0},
		{"(?i)trace_v[0-9]|missing", []string{"--regex"}, 1},
		{"hidden_[0-9]+", []string{"--regex"}, 0},
		{"hidden_[0-9]+", []string{"--regex", "--raw"}, 1},
	} {
		args := append([]string{"transcript", "search", tc.q, p, "--json"}, tc.flags...)
		out, e := run(args...)
		var got transcriptSearchResult
		if e != nil || json.Unmarshal([]byte(out), &got) != nil || got.Returned != tc.n {
			t.Fatalf("S2_REGEX FAIL: %v: %s %v", tc, out, e)
		}
	}
	out, e := run("transcript", "search", "[", p, "--regex", "--json")
	if e == nil || !strings.Contains(e.Error(), "invalid --regex") || strings.Contains(out, `"matches"`) {
		t.Fatalf("S2_REGEX FAIL: invalid regex published %s %v", out, e)
	}
	t.Log("S2_REGEX PASS")
}

func TestTranscriptSearchMultipleInputs(t *testing.T) {
	_, run := setupNotebook(t)
	dir := t.TempDir()
	a := filepath.Join(dir, "a.jsonl")
	b := filepath.Join(dir, "nested", "b.output")
	for _, p := range []string{a, b} {
		writeTranscriptFile(t, p, `{"type":"session","id":"session"}`+"\n"+`{"type":"message","id":"event","agentId":"A","message":{"role":"assistant","content":"needle"}}`+"\n")
	}
	writeTranscriptFile(t, filepath.Join(dir, "notes.txt"), "not a transcript")
	alias := filepath.Join(t.TempDir(), "alias.jsonl")
	if e := os.Symlink(a, alias); e != nil {
		t.Fatal(e)
	}
	out, e := run("transcript", "search", "needle", a, dir, b, alias, "--json")
	var got transcriptSearchResult
	if e != nil || json.Unmarshal([]byte(out), &got) != nil || got.Returned != 2 || got.Truncated || got.SkippedFiles != 1 {
		t.Fatalf("S3_INPUTS FAIL: %s %v", out, e)
	}
	if got.Matches[0].SourcePath != a || got.Matches[1].SourcePath != b || got.Matches[1].AgentID != "A" || got.Matches[1].EventID != "event" {
		t.Fatalf("S3_INPUTS FAIL: provenance/order %+v", got)
	}
	out, e = run("transcript", "search", "needle", dir, "--limit", "1", "--json")
	if e != nil || json.Unmarshal([]byte(out), &got) != nil || got.Returned != 1 || !got.Truncated {
		t.Fatalf("S4_GLOBAL FAIL: %s %v", out, e)
	}
	for _, args := range [][]string{
		{"transcript", "search", "needle", a, filepath.Join(dir, "missing"), "--json"},
		{"transcript", "search", "needle", dir, "--session", a, "--json"},
		{"transcript", "search", "needle", filepath.Join(dir, "notes.txt"), "--json"},
	} {
		out, e = run(args...)
		if e == nil || strings.Contains(out, `"matches"`) {
			t.Fatalf("S4_GLOBAL FAIL: partial success %s %v", out, e)
		}
	}
	dirAlias := filepath.Join(t.TempDir(), "directory-link")
	if err := os.Symlink(dir, dirAlias); err != nil {
		t.Fatal(err)
	}
	files, _, err := transcriptSearchInputs([]string{dirAlias})
	if err != nil || len(files) != 2 {
		t.Fatalf("S3_INPUTS FAIL: directory symlink %v %v", files, err)
	}
	t.Log("S3_INPUTS PASS")
	bad := filepath.Join(dir, "late.jsonl")
	writeTranscriptFile(t, bad, `{"type":"session","id":"bad"}`+"\n"+strings.Repeat("x", 16*1024*1024)+"\n")
	out, e = run("transcript", "search", "needle", a, bad, "--regex", "--limit", "1", "--json")
	if e == nil || !strings.Contains(e.Error(), bad) || !strings.Contains(e.Error(), "token too long") || strings.Contains(out, `"matches"`) {
		t.Fatalf("S4_GLOBAL FAIL: late read failure hidden: %s %v", out, e)
	}
	t.Log("S4_GLOBAL PASS")
}
