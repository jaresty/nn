package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// claudeCodeFixture writes a minimal claude-code transcript (no Pi records).
func claudeCodeFixture(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "session.jsonl")
	writeTranscriptFile(t, p,
		`{"type":"assistant","uuid":"r1","message":{"role":"assistant","content":[],"usage":{"input_tokens":10,"output_tokens":5}}}`+"\n"+
			`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"ok"}]}}`+"\n",
	)
	return p
}

func TestReviewAcceptsClaudeCodeSession(t *testing.T) {
	const a = "ASSERT_REVIEW_ACCEPTS_CLAUDE_CODE"
	path := claudeCodeFixture(t)
	p, e := buildReviewPage(path, "open-handoff", "canonical", "", 20, "")
	if e != nil {
		t.Fatalf("%s: review must not error on claude-code session: %v", a, e)
	}
	if p.Eligible != 0 {
		t.Fatalf("%s: claude-code has no rooms; eligible must be 0, got %d", a, p.Eligible)
	}
	t.Log(a + ": PASS")
}

func TestContextAcceptsClaudeCodeSession(t *testing.T) {
	const a = "ASSERT_CONTEXT_ACCEPTS_CLAUDE_CODE"
	path := claudeCodeFixture(t)
	_, e := buildTranscriptContext(path, "ROOT", 5, 1, "")
	if e != nil {
		t.Fatalf("%s: context must not error on claude-code session for ROOT: %v", a, e)
	}
	t.Log(a + ": PASS")
}

func TestTranscriptReviewRegistered(t *testing.T) {
	c := newTranscriptCmd(nil)
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetArgs([]string{"review", "--help"})
	err := c.Execute()
	if err != nil || !strings.Contains(out.String(), "open-handoff") {
		t.Fatalf("review command must expose conservative queues: err=%v output=%s", err, out.String())
	}
}
