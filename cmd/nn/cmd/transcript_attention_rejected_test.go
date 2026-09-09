package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/attention"
)

func rejectionResult(t *testing.T, id, text string) ledgerRecord {
	t.Helper()
	r := attentionRecord(t, 2, "toolResult", `[]`)
	m := map[string]any{"role": "toolResult", "toolCallId": id, "toolName": "bash", "isError": true, "content": []map[string]string{{"type": "text", "text": text}}}
	r.Record.Message, _ = json.Marshal(m)
	return r
}

const rejectionText = "Validation failed for tool \"bash\":\n  - command: must have required properties command\n\nReceived arguments:\n{}"

func TestAttentionRejectedCommand(t *testing.T) {
	for _, tc := range []struct {
		name     string
		change   func([]ledgerRecord) []ledgerRecord
		rejected int
	}{
		{"linked", func(r []ledgerRecord) []ledgerRecord { return r }, 1},
		{"no-result", func(r []ledgerRecord) []ledgerRecord { return r[:1] }, 0},
		{"wrong-call", func(r []ledgerRecord) []ledgerRecord { r[1] = rejectionResult(t, "other", rejectionText); return r }, 0},
		{"other-source", func(r []ledgerRecord) []ledgerRecord { r[1].Path = "/foreign.jsonl"; return r }, 0},
		{"ordinary-error", func(r []ledgerRecord) []ledgerRecord {
			r[1] = rejectionResult(t, "bad", "command: must have required properties command")
			return r
		}, 0},
		{"wrong-arguments", func(r []ledgerRecord) []ledgerRecord {
			r[1] = rejectionResult(t, "bad", strings.TrimSuffix(rejectionText, "{}")+`{"x":1}`)
			return r
		}, 0},
		{"duplicate-result", func(r []ledgerRecord) []ledgerRecord { return append(r, r[1]) }, 0},
		{"duplicate-call", func(r []ledgerRecord) []ledgerRecord { return append([]ledgerRecord{r[0]}, r...) }, 0},
		{"before-call", func(r []ledgerRecord) []ledgerRecord { return []ledgerRecord{r[1], r[0]} }, 0},
		{"not-error", func(r []ledgerRecord) []ledgerRecord {
			r[1].Record.Message = []byte(strings.Replace(string(r[1].Record.Message), `"isError":true`, `"isError":false`, 1))
			return r
		}, 0},
		{"wrong-tool", func(r []ledgerRecord) []ledgerRecord {
			r[1].Record.Message = []byte(strings.Replace(string(r[1].Record.Message), `"toolName":"bash"`, `"toolName":"write"`, 1))
			return r
		}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			records := tc.change([]ledgerRecord{attentionRecord(t, 1, "assistant", attentionCall("bad", "bash", `{}`)), rejectionResult(t, "bad", rejectionText)})
			m, _, ev, err := collectAttention(records, "available", "A", 100)
			if err != nil || m.Rejected != tc.rejected || m.Commands != 0 || m.Edits != 0 {
				t.Fatalf("REJECTION_%s FAIL: %+v %v", tc.name, m, err)
			}
			if (m.Unknown == 0) != (tc.rejected == 1) {
				t.Fatalf("REJECTION_%s FAIL: uncertainty %+v", tc.name, m)
			}
			if tc.rejected == 1 && !strings.Contains(ev[0].Summary, ledgerID(records[1].Path, 2, "A", "message")) {
				t.Fatal("linked evidence absent")
			}
			t.Logf("REJECTION_%s PASS", tc.name)
		})
	}
}

func TestAttentionRejectedRetention(t *testing.T) {
	_, execute := setupNotebook(t)
	path, id := attentionFixture(t, "pi")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []ledgerRecord{attentionRecord(t, 41, "assistant", attentionCall("bad", "bash", `{}`)), rejectionResult(t, "bad", rejectionText)} {
		if _, err = fmt.Fprintf(f, "{\"type\":\"message\",\"message\":%s}\n", r.Record.Message); err != nil {
			f.Close()
			t.Fatal(err)
		}
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := execute("transcript", "attention", path, "--agent", id, "--task", "implementation", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	var p attentionPage
	if err = json.Unmarshal([]byte(out), &p); err != nil {
		t.Fatal(err)
	}
	if p.MetricVersion != 2 || len(p.Rooms) != 1 || p.Rooms[0].Metrics.Rejected != 1 || p.Rooms[0].Metrics.Commands != 40 || p.Rooms[0].Result.Status != "match" {
		t.Fatalf("rejection not retained: %+v", p)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	replay, err := execute("transcript", "attention", path, "--snapshot", p.Snapshot, "--format", "json")
	if err != nil || replay != out {
		t.Fatalf("replay changed metrics: %v", err)
	}
}

func TestAttentionRejectionDoesNotHideExecution(t *testing.T) {
	records := []ledgerRecord{attentionRecord(t, 1, "assistant", attentionCall("bad", "bash", `{"command":"echo x"}`)), rejectionResult(t, "bad", rejectionText)}
	m, _, _, err := collectAttention(records, "available", "A", 100)
	if err != nil || m.Commands != 1 || m.Rejected != 0 || m.Unknown != 0 {
		t.Fatalf("valid call hidden: %+v %v", m, err)
	}
	records[0] = attentionRecord(t, 1, "assistant", attentionCall("bad", "bash", `{}`))
	m, w, _, err := collectAttention(records, "available", "A", 1)
	if err != nil || m.Rejected != 0 || m.Commands != 0 || w.Selected != 1 {
		t.Fatalf("out-of-window call counted: %+v", m)
	}
	records = append(records, attentionRecord(t, 3, "assistant", attentionCall("ok", "bash", `{"command":"echo x"}`)))
	m, _, _, err = collectAttention(records, "available", "A", 100)
	p, _ := attention.Builtin()
	result, e := p.Evaluate("A", "implementation", m)
	if err != nil || e != nil || m.Rejected != 1 || m.Commands != 1 || result.Status != "no_match" {
		t.Fatalf("rejected call poisoned ratio: %+v %+v %v %v", m, result, err, e)
	}
	m.Commands = 0
	result, e = p.Evaluate("A", "implementation", m)
	if e != nil || result.Status != "indeterminate" {
		t.Fatal("rejected-only window must keep zero denominator unknown")
	}
	m.Rejected = -1
	if _, e = p.Evaluate("A", "implementation", m); e == nil {
		t.Fatal("negative rejected count accepted")
	}
	m.Rejected = 2001
	if _, e = p.Evaluate("A", "implementation", m); e == nil {
		t.Fatal("rejected count bypasses cap")
	}
}
