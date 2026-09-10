package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func observeAttentionFixture(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	path, _ := attentionFixture(t, "pi")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, id := range []string{"A", "B", "C"} {
		if _, err = fmt.Fprintf(f, "\n"+`{"type":"custom","customType":"subagents:record","data":{"id":%q,"type":"worker","status":"completed"}}`+"\n", id); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 40; i++ {
		if _, err = fmt.Fprintf(f, `{"type":"message","agentId":"C","message":{"role":"assistant","content":[{"type":"toolCall","id":"c%d","name":"bash","arguments":{"command":"echo x"}}]}}`+"\n", i); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func TestObserveAttentionIndependentScope(t *testing.T) {
	p := observeAttentionFixture(t)
	text, err := contextCommand(t, "observe", p, "--task", "implementation")
	if err != nil {
		t.Fatal(err, text)
	}
	for _, s := range []string{"Selected attention IDs: ROOT, A, B, C", "triggered=2; not_triggered=0; needs_context=0; not_applicable=0; insufficient_evidence=2", "Metric version: 2", "Inspect evidence: nn transcript attention inspect", "## ROOT —", "## A —", "## B —"} {
		if !strings.Contains(text, s) {
			t.Fatalf("OBSERVE_ATTENTION_SCOPE FAIL: missing %q", s)
		}
	}
	if strings.Contains(text, "## C —") {
		t.Fatal("attention widened readable sample")
	}
	snap := strings.SplitN(strings.SplitN(text, "Snapshot: ", 2)[1], "\n", 2)[0]
	retained, err := loadAttention(snap)
	if err != nil {
		t.Fatal(err)
	}
	if retained.Page.Evaluated != 4 || retained.Page.Unevaluated != 0 || retained.Page.Rooms[3].ID != "C" || retained.Page.Rooms[3].Result.Status != "match" {
		t.Fatal("native retained provenance lost")
	}
	inspection, err := contextCommand(t, "attention", "inspect", snap, "--agent", "C")
	if err != nil || !strings.Contains(inspection, "Retained evidence excerpts") {
		t.Fatal(err, inspection)
	}
	t.Log("OBSERVE_ATTENTION_SCOPE PASS")
}

func TestObserveAttentionUnknownTask(t *testing.T) {
	p := observeAttentionFixture(t)
	text, err := contextCommand(t, "observe", p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "needs_context=2") || !strings.Contains(text, "Condition: match") || !strings.Contains(text, "\nSnapshot:") || strings.Contains(text, "not evaluated — task scope not established") {
		t.Fatal("OBSERVE_ATTENTION_TASK FAIL: absent task suppressed measured signals", text)
	}
	if !strings.Contains(text, "## ROOT —") {
		t.Fatal("observation lost")
	}
	t.Log("OBSERVE_ATTENTION_TASK PASS")
}

func TestObserveAttentionNativeStates(t *testing.T) {
	p := observeAttentionFixture(t)
	text, err := contextCommand(t, "observe", p, "--task", "research", "--attention-agent", "C")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "not_applicable=1") || !strings.Contains(text, "outside attention cohort: 3") {
		t.Fatal(text)
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		fmt.Fprintf(f, `{"type":"message","agentId":"C","message":{"role":"assistant","content":[{"type":"toolCall","id":"edit%d","name":"edit","arguments":{"path":"x.go"}}]}}`+"\n", i)
	}
	f.Close()
	text, err = contextCommand(t, "observe", p, "--task", "implementation", "--attention-agent", "C")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "triggered=0; not_triggered=1; needs_context=0") || !strings.Contains(text, "2 recognized edits / 40 commands") {
		t.Fatal("OBSERVE_ATTENTION_STATES FAIL", text)
	}
	t.Log("OBSERVE_ATTENTION_STATES PASS")
}

func TestObserveAttentionSelection(t *testing.T) {
	agents := []agent{{ID: "ROOT"}}
	for i := 24; i >= 0; i-- {
		agents = append(agents, agent{ID: fmt.Sprintf("a%02d", i), ParentID: "nested"})
	}
	ids, err := selectObserveAttention(agents, observeAttentionOptions{Limit: 20})
	if err != nil || len(ids) != 20 || ids[0] != "ROOT" || ids[19] != "a18" {
		t.Fatal(ids, err)
	}
	ids, err = selectObserveAttention(agents, observeAttentionOptions{IDs: []string{"a24", "ROOT"}, Limit: 20})
	if err != nil || len(ids) != 2 || ids[0] != "a24" {
		t.Fatal(ids, err)
	}
	p := observeAttentionFixture(t)
	text, err := contextCommand(t, "observe", p, "--task", "implementation", "--attention-limit", "1")
	if err != nil || !strings.Contains(text, "selected 1 of 4; outside attention cohort: 3") || !strings.Contains(text, "1 selected · 3 omitted") {
		t.Fatal(err, text)
	}
	for _, flags := range [][]string{{"--attention-limit", "0"}, {"--attention-limit", "21"}, {"--attention-agent", "A", "--attention-limit", "2"}, {"--attention-agent", "missing"}, {"--attention-agent", "A", "--attention-agent", "A"}, {"--attention-agent", ""}, {"--task", ""}} {
		if _, err := contextCommand(t, append([]string{"observe", p}, flags...)...); err == nil {
			t.Fatal("invalid accepted", flags)
		}
	}
}

func TestObserveAttentionErrorIsNotNoMatch(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	p := filepath.Join(t.TempDir(), "pi.jsonl")
	writeTranscriptFile(t, p, "{\"type\":\"session\"}\n"+fmt.Sprintf(`{"type":"message","message":{"role":"assistant","content":%q}}`, strings.Repeat("x", 1024*1024+1))+"\n")
	text, err := contextCommand(t, "observe", p, "--task", "implementation")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Outcome: error") || !strings.Contains(text, "error=1") || !strings.Contains(text, "## ROOT —") || !strings.Contains(text, "not_triggered=0") {
		t.Fatal("OBSERVE_ATTENTION_ERROR FAIL", text)
	}
	t.Log("OBSERVE_ATTENTION_ERROR PASS")
}
