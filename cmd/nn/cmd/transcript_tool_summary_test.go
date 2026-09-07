package cmd

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"unicode/utf8"
)

func toolSummaryFixture() []ledgerEvent {
	tool := func(kind, id, name, status, match string, ordinal int, size any) ledgerEvent {
		m := map[string]any{"name": name, "match_status": status, "matched_event_id": nil, "call_id": id}
		if match != "" {
			m["matched_event_id"] = match
			m["call_id"] = "pair"
		}
		if kind == "tool_call" {
			m["arguments_bytes"] = size
		} else {
			m["result_size"] = map[string]any{"text_characters": size, "text_bytes": size, "content_bytes": size}
		}
		return ledgerEvent{"kind": kind, "event_id": id, "ordinal": ordinal, "tools": m}
	}
	c := tool("tool_call", "c1", "bash", "matched", "r1", 1, 26)
	c["payload"] = json.RawMessage(`{"arguments":{"command":"nn read a.go"}}`)
	return []ledgerEvent{c, tool("tool_result", "r1", "", "matched", "c1", 2, 10), tool("tool_call", "c2", "read", "ambiguous", "", 3, nil), tool("tool_result", "r2", "read", "ambiguous", "", 4, 20), tool("tool_result", "r3", "", "missing", "", 5, nil), tool("tool_call", "c3", "", "unavailable", "", 6, 2), tool("tool_result", "r4", "", "unavailable", "", 7, 0), {"kind": "message", "event_id": "wrapper", "tools": map[string]any{"result_size": map[string]any{"text_characters": 999}}}}
}
func toolSummaryObject(t *testing.T, events []ledgerEvent, limit int, group string) map[string]any {
	t.Helper()
	b, err := buildToolSummary("fixture", "A", "pi", "available", events, limit, group, "")
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err = json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func TestTranscriptToolSummaryTotals(t *testing.T) {
	v := toolSummaryObject(t, toolSummaryFixture(), 5, "")
	s := v["stats"].(map[string]any)
	if s["calls"] != float64(3) || s["results"] != float64(4) {
		t.Fatal("ASSERT_TOOL_TOTALS: fail")
	}
	sizes := s["sizes"].(map[string]any)
	for name, total := range map[string]float64{"argument_bytes": 28, "result_text_characters": 30, "result_text_bytes": 30, "result_content_bytes": 30} {
		m := sizes[name].(map[string]any)
		if m["known_total"] != total || m["total"] != nil || m["status"] != "partial" || m["unknown_records"] != float64(1) {
			t.Fatal("ASSERT_TOOL_TOTALS: fail")
		}
	}
	t.Log("ASSERT_TOOL_TOTALS: pass")
}
func TestTranscriptToolSummaryUnknownZero(t *testing.T) {
	for _, tc := range []struct {
		events []ledgerEvent
		status string
		total  any
	}{{nil, "unavailable", nil}, {toolSummaryFixture()[4:5], "unavailable", nil}, {toolSummaryFixture()[6:7], "complete", float64(0)}} {
		s := toolSummaryObject(t, tc.events, 0, "")["stats"].(map[string]any)
		sizes, ok := s["sizes"].(map[string]any)
		if !ok {
			t.Fatal("ASSERT_TOOL_UNKNOWN: fail")
		}
		m := sizes["result_text_characters"].(map[string]any)
		if m["status"] != tc.status || m["total"] != tc.total {
			t.Fatal("ASSERT_TOOL_UNKNOWN: fail")
		}
	}
	t.Log("ASSERT_TOOL_UNKNOWN: pass")
}
func TestTranscriptToolSummaryRank(t *testing.T) {
	v := toolSummaryObject(t, toolSummaryFixture(), 2, "")
	a := v["largest_results"].([]any)
	if len(a) != 2 || v["results_omitted"] != float64(2) {
		t.Fatal("ASSERT_TOOL_RANK: fail")
	}
	if a[0].(map[string]any)["event_id"] != "r2" || a[1].(map[string]any)["event_id"] != "r1" {
		t.Fatal("ASSERT_TOOL_RANK: fail")
	}
	all := toolSummaryObject(t, toolSummaryFixture(), 5, "")["largest_results"].([]any)
	if all[2].(map[string]any)["event_id"] != "r4" || all[3].(map[string]any)["event_id"] != "r3" {
		t.Fatal("ASSERT_TOOL_RANK: fail")
	}
	tie := toolSummaryFixture()[3]
	tie["event_id"] = "tie"
	tie["ordinal"] = 99
	tied := toolSummaryObject(t, append(toolSummaryFixture(), tie), 2, "")["largest_results"].([]any)
	if tied[0].(map[string]any)["event_id"] != "r2" || tied[1].(map[string]any)["event_id"] != "tie" {
		t.Fatal("ASSERT_TOOL_RANK: fail — unstable tie")
	}
	t.Log("ASSERT_TOOL_RANK: pass")
}
func TestTranscriptToolSummaryJoins(t *testing.T) {
	v := toolSummaryObject(t, toolSummaryFixture(), 5, "tool")
	a := v["largest_results"].([]any)
	if len(a) != 4 {
		t.Fatal("ASSERT_TOOL_JOINS: fail")
	}
	if a[0].(map[string]any)["call"] != nil {
		t.Fatal("ASSERT_TOOL_JOINS: fail")
	}
	c := a[1].(map[string]any)["call"].(map[string]any)
	if c["event_id"] != "c1" || c["command"] != "nn read a.go" || a[1].(map[string]any)["tool"] != "bash" {
		t.Fatal("ASSERT_TOOL_JOINS: fail")
	}
	groups := v["groups"].([]any)
	if len(groups) != 3 {
		t.Fatal("ASSERT_TOOL_JOINS: fail")
	}
	if groups[0].(map[string]any)["tool"] != nil || groups[1].(map[string]any)["tool"] != "bash" {
		t.Fatal("ASSERT_TOOL_JOINS: fail")
	}
	for _, g := range groups {
		s := g.(map[string]any)["stats"].(map[string]any)
		if s["calls"] != float64(1) {
			t.Fatal("ASSERT_TOOL_JOINS: fail")
		}
	}
	t.Log("ASSERT_TOOL_JOINS: pass")
}
func TestTranscriptToolSummaryPreview(t *testing.T) {
	events := toolSummaryFixture()
	raw, _ := json.Marshal(map[string]any{"input": map[string]any{"command": strings.Repeat("界", 400)}})
	events[0]["payload"] = json.RawMessage(raw)
	a := toolSummaryObject(t, events, 5, "")["largest_results"].([]any)
	if len(a) != 4 {
		t.Fatal("ASSERT_TOOL_PREVIEW: fail")
	}
	c := a[1].(map[string]any)["call"].(map[string]any)
	for _, key := range []string{"command", "arguments_preview"} {
		s := c[key].(string)
		if len(s) > 512 || !utf8.ValidString(s) {
			t.Fatal("ASSERT_TOOL_PREVIEW: fail")
		}
	}
	if c["command_truncated"] != true || c["arguments_truncated"] != true || len(c["arguments_sha256"].(string)) != 64 {
		t.Fatal("ASSERT_TOOL_PREVIEW: fail")
	}
	t.Log("ASSERT_TOOL_PREVIEW: pass")
}
func TestTranscriptToolSummarySnapshot(t *testing.T) {
	events := toolSummaryFixture()
	v := toolSummaryObject(t, events, 5, "")
	snapshot, ok := v["snapshot"].(string)
	if !ok || len(snapshot) != 64 {
		t.Fatal("ASSERT_TOOL_SNAPSHOT: fail")
	}
	if _, err := buildToolSummary("fixture", "A", "pi", "available", events, 5, "", snapshot); err != nil {
		t.Fatal(err)
	}
	if _, err := buildToolSummary("fixture", "A", "pi", "available", events, 4, "", snapshot); err == nil {
		t.Fatal("ASSERT_TOOL_SNAPSHOT: fail")
	}
	events[0]["payload"] = json.RawMessage(`{"arguments":{"command":"nn read b.go"}}`)
	if _, err := buildToolSummary("fixture", "A", "pi", "available", events, 5, "", snapshot); err == nil {
		t.Fatal("ASSERT_TOOL_SNAPSHOT: fail")
	}
	setTail := func(tail string) {
		args, _ := json.Marshal(map[string]string{"command": strings.Repeat("x", 800) + tail})
		raw, _ := json.Marshal(map[string]json.RawMessage{"arguments": args})
		events[0]["payload"] = json.RawMessage(raw)
		events[0]["tools"].(map[string]any)["arguments_bytes"] = len(args)
	}
	setTail("A")
	snapshot = toolSummaryObject(t, events, 5, "")["snapshot"].(string)
	setTail("B")
	if _, err := buildToolSummary("fixture", "A", "pi", "available", events, 5, "", snapshot); err == nil {
		t.Fatal("ASSERT_TOOL_SNAPSHOT: fail — hidden argument tail unbound")
	}
	t.Log("ASSERT_TOOL_SNAPSHOT: pass")
}
func TestTranscriptToolSummaryLimits(t *testing.T) {
	for _, tc := range []struct {
		limit int
		group string
	}{{-1, ""}, {101, ""}, {5, "bad"}} {
		if _, err := buildToolSummary("f", "A", "pi", "available", nil, tc.limit, tc.group, ""); err == nil {
			t.Fatal("ASSERT_TOOL_LIMITS: fail")
		}
	}
	t.Log("ASSERT_TOOL_LIMITS: pass")
}
func TestTranscriptToolSummaryOverflow(t *testing.T) {
	e := toolSummaryFixture()[2:3]
	e[0]["tools"].(map[string]any)["arguments_bytes"] = int64(math.MaxInt64)
	if _, err := buildToolSummary("f", "A", "pi", "available", append(e, toolSummaryFixture()[5]), 5, "", ""); err == nil {
		t.Fatal("ASSERT_TOOL_OVERFLOW: fail")
	}
	t.Log("ASSERT_TOOL_OVERFLOW: pass")
}
func TestTranscriptToolSummaryBound(t *testing.T) {
	huge := toolSummaryFixture()
	huge[3]["tools"].(map[string]any)["name"] = strings.Repeat("x", 48000)
	if _, err := buildToolSummary("f", "A", "pi", "available", huge, 5, "", ""); err == nil {
		t.Fatal("ASSERT_TOOL_BOUND: fail")
	}
	t.Log("ASSERT_TOOL_BOUND: pass")
}
