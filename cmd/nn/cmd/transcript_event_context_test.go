package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func contextFixture(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "session.jsonl")
	writeTranscriptFile(t, p, `{"type":"session","id":"context-fixture"}
{"type":"message","id":"before","message":{"role":"user","content":"orientation"}}
{"type":"message","id":"call","message":{"role":"assistant","metadata":"OPAQUE_METADATA","content":[{"type":"thinking","thinking":"HIDDEN_REASONING","signature":"HIDDEN_SIGNATURE"},{"type":"text","text":"First plan"},{"type":"toolCall","id":"c1","name":"bash","arguments":{"command":"bar build craft","path":"a\\b"}}]}}
{"type":"message","id":"result","message":{"role":"toolResult","toolCallId":"c1","toolName":"bash","content":"finished successfully"}}
{"type":"message","id":"next","message":{"role":"assistant","content":"Continue after result"}}
{"type":"message","id":"gap","message":{"role":"user","content":"gap"}}
{"type":"message","id":"later","message":{"role":"assistant","content":"Continue later"}}
{"type":"message","id":"end","message":{"role":"user","content":"end"}}
`)
	return p
}

func contextCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	c := newTranscriptCmd(nil)
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs(args)
	err := c.Execute()
	return out.String(), err
}
func contextPage(t *testing.T, p string, flags ...string) ledgerPage {
	t.Helper()
	out, err := contextCommand(t, append([]string{"events", p, "ROOT"}, flags...)...)
	if err != nil {
		t.Fatal(err, out)
	}
	var page ledgerPage
	if err = json.Unmarshal([]byte(out), &page); err != nil {
		t.Fatal(err, out)
	}
	return page
}
func contextEvents(t *testing.T, p ledgerPage) []ledgerEvent {
	t.Helper()
	events := []ledgerEvent{}
	for _, raw := range p.Events {
		var e ledgerEvent
		if err := json.Unmarshal(raw, &e); err != nil {
			t.Fatal(err)
		}
		events = append(events, e)
	}
	return events
}

func TestTranscriptEventContextCallAndContinuation(t *testing.T) {
	p := contextFixture(t)
	page := contextPage(t, p, "--kind", "tool_call", "--search", "BAR BUILD", "-B", "1", "-A", "3")
	if len(page.Events) != 5 || page.Query.Window.SelectedMatches != 1 {
		t.Fatalf("bad window: %+v", page)
	}
	es := contextEvents(t, page)
	if es[1]["kind"] != "tool_call" || es[1]["window_match"] != true || es[4]["kind"] != "message" || es[4]["window_match"] != false {
		t.Fatal(es)
	}
	for _, e := range es {
		if _, ok := e["payload"]; ok {
			t.Fatal("payload leaked")
		}
	}
	call := es[1]["event_id"].(string)
	exact := contextPage(t, p, "--event", call, "-C", "1")
	if len(exact.Events) != 3 || exact.Query.Window.Anchor != call {
		t.Fatal(exact)
	}
	text, err := contextCommand(t, "events", p, "ROOT", "--event", call, "-A", "3", "--format", "text")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"MATCH CALL", "CONTEXT RESULT", "Continue after result"} {
		if !strings.Contains(text, s) {
			t.Fatal(s, text)
		}
	}
	if strings.Contains(text, "HIDDEN") || strings.Contains(text, "OPAQUE") {
		t.Fatal("opaque content leaked", text)
	}
}

func TestTranscriptEventContextFiltersAndGaps(t *testing.T) {
	p := contextFixture(t)
	for _, query := range []string{"HIDDEN_REASONING", "HIDDEN_SIGNATURE", "OPAQUE_METADATA"} {
		page := contextPage(t, p, "--search", query)
		if len(page.Events) != 0 {
			t.Fatal("non-readable match", query)
		}
	}
	for _, tt := range []struct {
		flags []string
		n     int
	}{
		{[]string{"--search", "bar.*craft"}, 0},
		{[]string{"--search", "bar.*craft", "--regex"}, 1},
		{[]string{"--role", "assistant", "--search", "continue"}, 2},
		{[]string{"--role", "user", "--search", "continue"}, 0},
		{[]string{"--kind", "tool_result", "--search", "finished"}, 1},
		{[]string{"--kind", "tool_call", "--search", `a\b`}, 1},
	} {
		page := contextPage(t, p, tt.flags...)
		if len(page.Events) != tt.n {
			t.Fatalf("%v: %d != %d", tt.flags, len(page.Events), tt.n)
		}
	}
	merged := contextPage(t, p, "--search", "continue", "-C", "1")
	if merged.Query.Window.Windows != 1 || len(merged.Events) != 5 {
		t.Fatal(merged)
	}
	limited := contextPage(t, p, "--search", "continue", "--max-matches", "1", "--last", "1")
	if limited.Query.Window.OmittedMatches != 1 || contextEvents(t, limited)[0]["ordinal"] != float64(8) {
		t.Fatal(limited)
	}
	text, err := contextCommand(t, "events", p, "ROOT", "--search", "continue", "--format", "text")
	if err != nil || !strings.Contains(text, "-- 1 events omitted --") {
		t.Fatal(err, text)
	}
}

func TestTranscriptEventContextSnapshotsAndValidation(t *testing.T) {
	p := contextFixture(t)
	page := contextPage(t, p, "--kind", "tool_call", "-A", "2", "--select", "identity")
	for _, flags := range [][]string{
		{"--kind", "bad"}, {"--search", ""}, {"--regex"}, {"--search", "[", "--regex"},
		{"-C", "1", "-A", "0"}, {"-C", "-1"}, {"-A", "201"}, {"--max-matches", "0"},
		{"--search", "plan", "--summary", "tools"}, {"--event", "missing", "-C", "2"},
		{"--event", "missing", "--search", "x"}, {"--search", "x", "--format", "text", "--last", "0"},
		{"--kind", "message", "-A", "2", "--select", "identity", "--snapshot", page.Snapshot},
	} {
		if _, err := contextCommand(t, append([]string{"events", p, "ROOT"}, flags...)...); err == nil {
			t.Fatal("invalid accepted", flags)
		}
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("{\"type\":\"message\",\"message\":{\"role\":\"user\",\"content\":\"new\"}}\n")
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = contextCommand(t, "events", p, "ROOT", "--kind", "tool_call", "-A", "2", "--select", "identity", "--snapshot", page.Snapshot); err == nil {
		t.Fatal("stale snapshot accepted")
	}
}

func TestTranscriptSearchContextSymmetryAndIsolation(t *testing.T) {
	p := contextFixture(t)
	q := contextFixture(t)
	out, err := contextCommand(t, "search", "continue", p, q, "-C", "1", "--json")
	if err != nil {
		t.Fatal(err, out)
	}
	var r transcriptSearchResult
	if err = json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatal(err)
	}
	if len(r.Context) != 2 || r.Returned != 4 {
		t.Fatal(out)
	}
	for _, c := range r.Context {
		if len(c.Events) != 5 || c.Query.Window.Windows != 1 {
			t.Fatal(out)
		}
	}
	for _, m := range r.Matches {
		if len(m.ContextEventID) != 64 {
			t.Fatal("missing canonical anchor", m)
		}
	}
	if strings.Contains(out, "HIDDEN") || strings.Contains(out, "OPAQUE") {
		t.Fatal("opaque context leaked", out)
	}
	text, err := contextCommand(t, "search", "First plan", p, "-A", "4")
	if err != nil || !strings.Contains(text, "Continue after result") || !strings.Contains(text, "CONTEXT tool_call") {
		t.Fatal(err, text)
	}
	plain, err := contextCommand(t, "search", "continue", p, "--json")
	if err != nil || strings.Contains(plain, "context_event_id") || strings.Contains(plain, "window_snapshot") {
		t.Fatal("legacy changed", err, plain)
	}
	for _, flags := range [][]string{{"-C", "-1"}, {"-C", "1", "-B", "0"}, {"-C", "1", "--limit", "201"}} {
		if _, err := contextCommand(t, append([]string{"search", "x", p}, flags...)...); err == nil {
			t.Fatal(flags)
		}
	}
}
