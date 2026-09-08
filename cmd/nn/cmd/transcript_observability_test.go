package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func observabilityRecords(t *testing.T) []ledgerRecord {
	t.Helper()
	lines := []string{
		`{"type":"message","timestamp":"2026-09-08T11:52:13Z","message":{"role":"assistant","timestamp":1788868333000,"content":[{"type":"toolCall","id":"c","name":"Bash","arguments":{}}]}}`,
		`{"type":"message","timestamp":"2026-09-08T11:52:15Z","message":{"role":"toolResult","toolCallId":"c","isError":true,"content":"PRIVATE RESULT"}}`,
		`{"type":"message","timestamp":"2026-09-08T12:24:30Z","message":{"role":"assistant","timestamp":1788868335000,"stopReason":"error","errorMessage":"terminated","content":[{"type":"thinking","thinking":"PRIVATE THINKING"}]}}`,
		`{"type":"message","message":{"role":"assistant","content":"no clock"}}`,
		`{"type":"message","timestamp":"bad-clock","message":{"role":"assistant","timestamp":1788868335000,"content":"invalid record clock"}}`,
		`{"type":"message","timestamp":"2026-09-08T12:24:29Z","message":{"role":"user","content":"earlier"}}`,
		`{"type":"message","timestamp":"2026-09-08T12:24:28Z","message":{"role":"assistant","content":"backwards"}}`,
	}
	var out []ledgerRecord
	for i, line := range lines {
		var r rawRecord
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatal(err)
		}
		r.RecordOrdinal = i + 1
		out = append(out, ledgerRecord{Record: r, Path: "/fixture.jsonl"})
	}
	return out
}

func observabilitySession(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "session.jsonl")
	text := "{\"type\":\"session\"}\n"
	for _, r := range observabilityRecords(t) {
		b, _ := json.Marshal(r.Record)
		text += string(b) + "\n"
	}
	writeTranscriptFile(t, path, text)
	return path
}

func TestTranscriptObservabilityTiming(t *testing.T) {
	const a = "ASSERT_OBSERVABILITY_TIMING"
	_, execute := setupNotebook(t)
	out, err := execute("transcript", "events", observabilitySession(t), "ROOT", "--summary", "timing", "--limit", "2")
	if err != nil {
		t.Fatalf("%s: %v", a, err)
	}
	var s struct {
		Version  string         `json:"version"`
		Messages int            `json:"message_records"`
		Clocks   map[string]int `json:"timestamps"`
		Gaps     struct {
			Intervals int     `json:"intervals"`
			Unknown   int     `json:"unknown_intervals"`
			Negative  int     `json:"negative_intervals"`
			Seconds   float64 `json:"observed_seconds"`
		} `json:"message_gaps"`
		Tools struct {
			Intervals int     `json:"intervals"`
			Seconds   float64 `json:"observed_seconds"`
		} `json:"tool_intervals"`
		Largest []struct {
			Seconds float64 `json:"seconds"`
			From    string  `json:"from_event_id"`
			To      string  `json:"to_event_id"`
		} `json:"largest_gaps"`
		Errors map[string]int `json:"errors"`
	}
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatal(err)
	}
	if s.Version != "nn.transcript.timing-summary/v1" || s.Messages != 7 || s.Clocks["known"] != 5 || s.Clocks["missing"] != 1 || s.Clocks["invalid"] != 1 || s.Gaps.Intervals != 3 || s.Gaps.Unknown != 3 || s.Gaps.Negative != 1 || s.Gaps.Seconds != 1937 || s.Tools.Intervals != 1 || s.Tools.Seconds != 2 || len(s.Largest) != 2 || s.Largest[0].Seconds != 1935 || s.Largest[0].From == s.Largest[0].To || s.Errors["assistant_errors"] != 1 || s.Errors["tool_errors"] != 1 {
		t.Fatalf("%s: %s", a, out)
	}
	t.Log(a + ": PASS")
}

func TestTranscriptObservabilitySelection(t *testing.T) {
	const a = "ASSERT_OBSERVABILITY_SELECTION"
	_, execute := setupNotebook(t)
	session := observabilitySession(t)
	out, err := execute("transcript", "events", session, "ROOT", "--since", "2026-09-08T11:52:15Z", "--until", "2026-09-08T12:24:30Z", "--select", "identity")
	if err != nil {
		t.Fatalf("%s: %v", a, err)
	}
	var p struct {
		Events []map[string]any `json:"events"`
		Query  map[string]any   `json:"query"`
	}
	_ = json.Unmarshal([]byte(out), &p)
	if len(p.Events) != 5 || p.Events[0]["ordinal"] != float64(3) || p.Events[4]["ordinal"] != float64(9) || p.Query["excluded_unknown_timestamp"] != float64(2) || p.Query["total_events"] != float64(9) {
		t.Fatalf("%s: %s", a, out)
	}
	for _, e := range p.Events {
		if _, ok := e["message"]; ok {
			t.Fatalf("%s: selection leaked facet", a)
		}
	}
	out, err = execute("transcript", "events", session, "ROOT", "--errors-only", "--select", "identity")
	if err != nil {
		t.Fatalf("%s: %v", a, err)
	}
	_ = json.Unmarshal([]byte(out), &p)
	if len(p.Events) != 2 || p.Events[0]["kind"] != "tool_result" || p.Events[1]["ordinal"] != float64(5) {
		t.Fatalf("%s: %s", a, out)
	}
	for _, flags := range [][]string{{"--since", "bad"}, {"--since", ""}, {"--since", "2026-09-08T12:24:30Z", "--until", "2026-09-08T11:52:15Z"}, {"--summary", "timing", "--since", "2026-09-08T00:00:00Z"}, {"--errors-only", "--event", "unknown"}} {
		if _, err := execute(append([]string{"transcript", "events", session, "ROOT"}, flags...)...); err == nil {
			t.Fatalf("%s: accepted %v", a, flags)
		}
	}
	t.Log(a + ": PASS")
}

func TestTranscriptObservabilityTransport(t *testing.T) {
	const a = "ASSERT_OBSERVABILITY_TRANSPORT"
	_, execute := setupNotebook(t)
	session := observabilitySession(t)
	// Make the selected error event larger than one page without affecting selection.
	bytes, err := os.ReadFile(session)
	if err != nil {
		t.Fatal(err)
	}
	bytes = []byte(strings.Replace(string(bytes), "terminated", strings.Repeat("α😀", 18000), 1))
	writeTranscriptFile(t, session, string(bytes))
	flags := []string{"transcript", "events", session, "ROOT", "--since", "2026-09-08T12:24:30Z", "--until", "2026-09-08T12:24:30Z", "--payload"}
	out, err := execute(flags...)
	if err != nil {
		t.Fatalf("%s: %v", a, err)
	}
	var p ledgerPage
	if err := json.Unmarshal([]byte(out), &p); err != nil {
		t.Fatal(err)
	}
	if p.Pages < 2 || len(out) > 48000 {
		t.Fatalf("%s: expected bounded multi-page event", a)
	}
	snapshot := p.Snapshot
	var reconstructed strings.Builder
	for {
		for _, raw := range p.Events {
			var f struct {
				Text string `json:"text"`
			}
			_ = json.Unmarshal(raw, &f)
			reconstructed.WriteString(f.Text)
		}
		if p.NextPage == 0 {
			break
		}
		out, err = execute(append(flags, "--page", fmt.Sprint(p.NextPage), "--snapshot", snapshot)...)
		if err != nil || len(out) > 48000 {
			t.Fatalf("%s: page: %v", a, err)
		}
		_ = json.Unmarshal([]byte(out), &p)
	}
	var e map[string]any
	if json.Unmarshal([]byte(reconstructed.String()), &e) != nil || e["ordinal"] != float64(5) {
		t.Fatalf("%s: fragment reconstruction", a)
	}
	out, err = execute(append(flags, "--all")...)
	if err != nil {
		t.Fatalf("%s: %v", a, err)
	}
	var all ledgerPage
	_ = json.Unmarshal([]byte(out), &all)
	if all.Snapshot != snapshot || len(all.Events) != 1 || string(all.Events[0]) != reconstructed.String() {
		t.Fatalf("%s: complete export differs", a)
	}
	writeTranscriptFile(t, session, strings.Replace(string(bytes), "no clock", "changed outside window", 1))
	if _, err := execute(append(flags, "--snapshot", snapshot)...); err == nil {
		t.Fatalf("%s: accepted changed source", a)
	}
	t.Log(a + ": PASS")
}

func TestTranscriptObservabilityTimingEdges(t *testing.T) {
	const a = "ASSERT_OBSERVABILITY_TIMING"
	run := func(records []ledgerRecord, limit int) map[string]any {
		t.Helper()
		es, err := projectLedger(records, "ROOT", []string{"identity", "message", "tools"}, false)
		if err != nil {
			t.Fatal(err)
		}
		b, err := buildTimingSummary("/fixture", "ROOT", "pi", "available", es, limit, "")
		if err != nil {
			t.Fatalf("%s: %v", a, err)
		}
		var out map[string]any
		_ = json.Unmarshal(b, &out)
		return out
	}
	empty := run(nil, 0)
	g := empty["message_gaps"].(map[string]any)
	if g["total_seconds"] != nil || g["status"] != "unavailable" || empty["elapsed_seconds"] != nil {
		t.Fatalf("%s: empty must not be measured zero: %v", a, empty)
	}
	rs := observabilityRecords(t)[:2]
	rs[1].Record.Timestamp = rs[0].Record.Timestamp
	zero := run(rs, 0)
	g = zero["message_gaps"].(map[string]any)
	if g["total_seconds"] != float64(0) || g["status"] != "complete" || zero["gaps_omitted"] != float64(1) {
		t.Fatalf("%s: measured zero: %v", a, zero)
	}
	rs = append(rs, rs[0])
	rs[2].Record.RecordOrdinal = 100
	dupe := run(rs, 1)
	if dupe["tool_result_joins"].(map[string]any)["ambiguous"] != float64(1) || dupe["tool_intervals"].(map[string]any)["intervals"] != float64(0) {
		t.Fatalf("%s: duplicate join: %v", a, dupe)
	}
	ts, status := ledgerTime(json.RawMessage(`1000`))
	if status != "known" || ts.Unix() != 1 {
		t.Fatalf("%s: milliseconds interpreted incorrectly", a)
	}
	for _, v := range []any{"bad", json.RawMessage(`1.5`), json.RawMessage(`9223372036854775807`)} {
		if _, status := ledgerTime(v); status != "invalid" {
			t.Fatalf("%s: invalid clock %v", a, v)
		}
	}
	rs = observabilityRecords(t)[:2]
	rs[0].Record.Timestamp = "0001-01-01T00:00:00Z"
	rs[1].Record.Timestamp = "9999-01-01T00:00:00Z"
	es, _ := projectLedger(rs, "ROOT", []string{"identity", "message", "tools"}, false)
	if _, err := buildTimingSummary("/fixture", "ROOT", "pi", "available", es, 1, ""); err == nil {
		t.Fatalf("%s: saturated interval accepted", a)
	}
	es, _ = projectLedger(observabilityRecords(t), "ROOT", []string{"identity", "message", "tools"}, false)
	b, err := buildTimingSummary("/fixture", "ROOT", "pi", "available", es, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	_ = json.Unmarshal(b, &doc)
	if _, err := buildTimingSummary("/fixture", "ROOT", "pi", "available", es, 1, doc["snapshot"].(string)); err != nil {
		t.Fatalf("%s: stable snapshot: %v", a, err)
	}
	if _, err := buildTimingSummary("/fixture", "ROOT", "pi", "available", es, 2, doc["snapshot"].(string)); err == nil {
		t.Fatalf("%s: mismatched summary snapshot accepted", a)
	}
	// Pathological labels are not silently truncated to fit the summary.
	es[0]["message"].(map[string]any)["role"] = strings.Repeat("x", 50000)
	if _, err := buildTimingSummary("/fixture", "ROOT", "pi", "available", es, 0, ""); err == nil {
		t.Fatalf("%s: oversized summary accepted", a)
	}
	t.Log(a + ": PASS")
}

func TestTranscriptObservabilitySkill(t *testing.T) {
	const a = "ASSERT_OBSERVABILITY_SKILL"
	for path, phrases := range map[string][]string{
		"references/summaries.md": {"--summary timing", "not execution time"},
		"references/events.md":    {"--errors-only", "--since", "--until"},
	} {
		body, err := os.ReadFile(filepath.Join("..", "..", "..", "skills", "nn-transcript", path))
		if err != nil {
			t.Fatal(err)
		}
		for _, phrase := range phrases {
			if !strings.Contains(string(body), phrase) {
				t.Fatalf("%s: %s lacks %s", a, path, phrase)
			}
		}
	}
	t.Log(a + ": PASS")
}

func TestTranscriptObservabilityMetadata(t *testing.T) {
	const a = "ASSERT_OBSERVABILITY_METADATA"
	es, err := projectLedger(observabilityRecords(t), "ROOT", []string{"identity", "message", "tools"}, false)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(es[4]["message"])
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["stop_reason"] != "error" || m["error_message"] != "terminated" || m["record_timestamp"] != "2026-09-08T12:24:30Z" || m["message_timestamp"] != float64(1788868335000) {
		t.Fatalf("%s: %s", a, b)
	}
	b, _ = json.Marshal(es[5]["message"])
	_ = json.Unmarshal(b, &m)
	for _, k := range []string{"stop_reason", "error_message", "record_timestamp", "message_timestamp"} {
		v, ok := m[k]
		if !ok || v != nil {
			t.Fatalf("%s: unknown %s must be null: %s", a, k, b)
		}
	}
	if _, ok := es[4]["payload"]; ok {
		t.Fatalf("%s: native payload leaked", a)
	}
	t.Log(a + ": PASS")
}

func TestTranscriptObservabilityReadable(t *testing.T) {
	const a = "ASSERT_OBSERVABILITY_READABLE"
	var recs []rawRecord
	for _, r := range observabilityRecords(t) {
		recs = append(recs, r.Record)
	}
	var b strings.Builder
	renderMeaningfulEvents(&b, recs)
	if !strings.Contains(b.String(), `[failure stop_reason="error" error_message="terminated"]`) || strings.Contains(b.String(), "PRIVATE") {
		t.Fatalf("%s: %s", a, b.String())
	}
	var alias rawRecord
	_ = json.Unmarshal([]byte(`{"type":"message","message":{"role":"assistant","stopReason":"","stop_reason":"error","content":"visible"}}`), &alias)
	b.Reset()
	renderMeaningfulEvents(&b, []rawRecord{alias})
	if strings.Contains(b.String(), "[failure") {
		t.Fatalf("%s: alias precedence differs from metadata", a)
	}
	_ = json.Unmarshal([]byte(`{"type":"message","message":{"role":"assistant","stop_reason":"aborted","content":"visible"}}`), &alias)
	b.Reset()
	renderMeaningfulEvents(&b, []rawRecord{alias})
	if !strings.Contains(b.String(), `[failure stop_reason="aborted"`) {
		t.Fatalf("%s: snake_case abort hidden", a)
	}
	t.Log(a + ": PASS")
}
