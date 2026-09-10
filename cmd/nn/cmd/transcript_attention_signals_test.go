package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jaresty/nn/internal/attention"
)

func TestAttentionAllSignalsAndErrorIsolation(t *testing.T) {
	_, execute := setupNotebook(t)
	path, id := attentionFixture(t, "pi")
	first, err := attention.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	second := *first
	second.ID = "test-only-second"
	second.Window.LastWorkEvents = 1
	r, err := buildAttentionPolicies(path, []string{id}, "implementation", nil, []*attention.Policy{first, &second})
	if err != nil {
		t.Fatal(err)
	}
	s := r.Page.Rooms[0].Signals
	if r.Page.Version != 2 || r.Page.SignalEvaluations != 2 || len(s) != 2 || s[0].Outcome != "triggered" || s[1].Outcome != "not_triggered" || s[1].Metrics.Commands != 1 {
		t.Fatalf("ALL_SIGNALS FAIL: %+v", s)
	}
	// A larger window encounters an oversized earlier message; the smaller one
	// still evaluates. Neither parser/build errors nor one signal's failure hide it.
	recs := []ledgerRecord{attentionRecord(t, 1, "assistant", `"`+strings.Repeat("x", 1024*1024)+`"`), attentionRecord(t, 2, "assistant", attentionCall("last", "bash", `{"command":"echo x"}`))}
	signals, evidence := collectAttentionSignals([]*attention.Policy{first, &second}, recs, "available", "A", "implementation", "cohort_override", nil)
	if signals[0].Outcome != "error" || signals[1].Outcome != "not_triggered" || len(evidence) != 1 {
		t.Fatalf("SIGNAL_ISOLATION FAIL: %+v", signals)
	}
	out, err := execute("transcript", "attention", path, "--snapshot", r.Page.Snapshot, "--format", "json")
	if err != nil || !strings.Contains(out, "test-only-second") {
		t.Fatal(err, out)
	}
	t.Log("ALL_SIGNALS PASS; SIGNAL_ISOLATION PASS")
}

func TestAttentionAutomaticAndMixedTasks(t *testing.T) {
	p := observeAttentionFixture(t)
	out, err := contextCommand(t, "attention", p, "--agent", "ROOT", "--agent", "C", "--agent-task", "C=implementation", "--format", "json")
	if err != nil {
		t.Fatal(err, out)
	}
	var page attentionPage
	if err = json.Unmarshal([]byte(out), &page); err != nil {
		t.Fatal(err)
	}
	if page.SelectedAgents != 2 || page.Rooms[0].Signals[0].Outcome != "needs_context" || page.Rooms[0].Signals[0].Condition.Status != "match" || page.Rooms[1].Signals[0].Outcome != "triggered" || page.Rooms[1].Signals[0].Applicability.Source != "agent_override" {
		t.Fatal("AUTOMATIC_SIGNALS FAIL", out)
	}
	replay, err := contextCommand(t, "attention", p, "--snapshot", page.Snapshot, "--format", "json")
	if err != nil || replay != out {
		t.Fatal("new replay changed", err)
	}
	if _, err = contextCommand(t, "attention", p, "--snapshot", page.Snapshot, "--agent-task", "C=research"); err == nil {
		t.Fatal("override mismatch accepted")
	}
	for _, flags := range [][]string{{"--agent-task", "C=implementation"}, {"--agent-task", "ROOT="}, {"--agent-task", "ROOT=research", "--agent-task", "ROOT=implementation"}} {
		if _, err := contextCommand(t, append([]string{"attention", p, "--agent", "ROOT"}, flags...)...); err == nil {
			t.Fatal("invalid override accepted", flags)
		}
	}
	t.Log("AUTOMATIC_SIGNALS PASS")
}

func TestAttentionTaskEvidence(t *testing.T) {
	prompt := strings.Repeat("Implement λ safely. ", 100)
	var call, ack rawRecord
	callJSON := `{"type":"message","message":{"role":"assistant","content":[{"type":"toolCall","id":"launch","name":"Agent","arguments":{"prompt":` + mustJSONString(t, prompt) + `,"signature":"SECRET"}}]}}`
	ackJSON := `{"type":"message","message":{"role":"toolResult","toolName":"Agent","toolCallId":"launch","details":{"status":"background","agentId":"A"}}}`
	if err := json.Unmarshal([]byte(callJSON), &call); err != nil {
		t.Fatal(err)
	}
	call.RecordOrdinal = 1
	if err := json.Unmarshal([]byte(ackJSON), &ack); err != nil {
		t.Fatal(err)
	}
	ack.RecordOrdinal = 2
	context := attentionTaskContextFor("A", "/captured/parent", nil, piHandoffs([]rawRecord{call, ack}))
	if context.Source != "authenticated_launches" || context.Status != "partial" || len(context.Evidence) != 1 || context.Evidence[0].OmittedBytes == 0 || len(context.Evidence[0].Text) > 1024 || !utf8.ValidString(context.Evidence[0].Text) || strings.Contains(context.Evidence[0].Text, "SECRET") {
		t.Fatalf("TASK_EVIDENCE FAIL: %+v", context)
	}
	if context.Evidence[0].EventID != ledgerID("/captured/parent", 1, "ROOT", "block:0") {
		t.Fatal("foreign identity")
	}
	ambiguous := attentionTaskContextFor("A", "/captured/parent", nil, piHandoffs([]rawRecord{call, call, ack}))
	if ambiguous.Status != "unavailable" || ambiguous.Unavailable != 1 || len(ambiguous.Evidence) != 0 {
		t.Fatal("ambiguous prompt adopted", ambiguous)
	}
	owned := []ledgerRecord{}
	for i := 0; i < 4; i++ {
		owned = append(owned, attentionRecord(t, i+1, "user", `"Implement something"`))
	}
	context = attentionTaskContextFor("A", "/captured/parent", owned, nil)
	if context.Total != 4 || context.Omitted != 2 || len(context.Evidence) != 2 || context.Evidence[0].Ordinal != 3 || context.Governing != "not_inferred" {
		t.Fatal(context)
	}
	t.Log("TASK_EVIDENCE PASS")
}

func TestAttentionLegacyCaptureReplay(t *testing.T) {
	_, execute := setupNotebook(t)
	path, id := attentionFixture(t, "claude-code")
	r, err := buildAttention(path, []string{id}, "implementation")
	if err != nil {
		t.Fatal(err)
	}
	// A historical v1 capture has no new plural/context/override fields.
	r.Page.Version = 1
	r.Page.SelectedAgents = 0
	r.Page.Policies = nil
	r.Page.AgentTasks = nil
	r.Page.SignalEvaluations = 0
	r.Page.Snapshot = ""
	for i := range r.Page.Rooms {
		r.Page.Rooms[i].Signals = nil
		r.Page.Rooms[i].TaskContext = nil
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := captureHash(b)
	if err = writeCaptureFile(snapshot+".attention", b); err != nil {
		t.Fatal(err)
	}
	r.Page.Snapshot = snapshot
	expected, _ := json.Marshal(r.Page)
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	out, err := execute("transcript", "attention", path, "--snapshot", snapshot, "--format", "json")
	if err != nil || strings.TrimSpace(out) != string(expected) {
		t.Fatal("LEGACY_REPLAY FAIL", err, out)
	}
	inspected, err := execute("transcript", "attention", "inspect", snapshot, "--agent", id)
	if err != nil || !strings.Contains(inspected, "echo x") {
		t.Fatal(err, inspected)
	}
	t.Log("LEGACY_REPLAY PASS")
}
