package cmd

import (
	"encoding/json"
	"testing"
)

func summaryFixture(t *testing.T) []ledgerEvent {
	t.Helper()
	rows := []ledgerEvent{}
	for i, raw := range []string{`{"input":10,"output":2,"cacheRead":30,"cacheWrite":0}`, `{"input":0,"output":0,"cacheRead":0,"cacheWrite":0}`, `{"input":5}`, `{}`} {
		u, err := ledgerUsage(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, ledgerEvent{"kind": "message", "event_id": fmtInt(i + 1), "usage": u})
	}
	rows = append(rows, ledgerEvent{"kind": "tool_call", "event_id": "tool", "usage": rows[0]["usage"]})
	return rows
}
func summaryValue(t *testing.T, events []ledgerEvent, size int) map[string]any {
	t.Helper()
	v, err := summarizeLedgerUsage(events, size)
	if err != nil {
		t.Fatal(err)
	}
	return exportObject(t, v)
}
func TestTranscriptUsageSummaryTotals(t *testing.T) {
	v := summaryValue(t, summaryFixture(t), 0)
	stats, ok := v["stats"].(map[string]any)
	if !ok {
		t.Fatal("ASSERT_USAGE_TOTALS: fail")
	}
	totals := stats["totals"].(map[string]any)
	if stats["records"] != float64(4) || stats["known_total_tokens"] != float64(47) || stats["total_tokens"] != nil || totals["input_tokens"] != float64(15) || totals["cache_read_tokens"] != float64(30) {
		t.Fatal("ASSERT_USAGE_TOTALS: fail")
	}
	t.Log("ASSERT_USAGE_TOTALS: pass")
}
func TestTranscriptUsageSummaryAuthority(t *testing.T) {
	v := summaryValue(t, summaryFixture(t), 0)
	s, ok := v["stats"].(map[string]any)
	if !ok {
		t.Fatal("ASSERT_USAGE_AUTHORITY: fail")
	}
	if s["status"] != "partial" || s["complete_records"] != float64(2) || s["partial_records"] != float64(1) || s["unavailable_records"] != float64(1) || s["zero_usage_records"] != float64(1) {
		t.Fatal("ASSERT_USAGE_AUTHORITY: fail")
	}
	missing := s["missing_counts"].(map[string]any)
	if missing["input_tokens"] != float64(1) || missing["output_tokens"] != float64(2) {
		t.Fatal("ASSERT_USAGE_AUTHORITY: fail")
	}
	empty := summaryValue(t, nil, 0)["stats"].(map[string]any)
	if empty["status"] != "unavailable" || empty["total_tokens"] != nil {
		t.Fatal("ASSERT_USAGE_AUTHORITY: fail")
	}
	for _, events := range [][]ledgerEvent{nil, summaryFixture(t)[3:4]} {
		unknown := summaryValue(t, events, 0)["stats"].(map[string]any)
		for _, value := range unknown["totals"].(map[string]any) {
			if value != nil {
				t.Fatal("ASSERT_USAGE_AUTHORITY: fail — unknown component became zero")
			}
		}
	}
	zero := summaryValue(t, summaryFixture(t)[1:2], 0)["stats"].(map[string]any)
	if zero["status"] != "complete" || zero["total_tokens"] != float64(0) {
		t.Fatal("ASSERT_USAGE_AUTHORITY: fail — measured zero lost")
	}
	t.Log("ASSERT_USAGE_AUTHORITY: pass")
}
func TestTranscriptUsageSummaryContext(t *testing.T) {
	v := summaryValue(t, summaryFixture(t), 0)
	s, ok := v["stats"].(map[string]any)
	if !ok {
		t.Fatal("ASSERT_USAGE_CONTEXT: fail")
	}
	c := s["context"].(map[string]any)
	if c["known_records"] != float64(2) || c["unknown_records"] != float64(2) || c["zero_records"] != float64(1) || c["first"] != float64(40) || c["last"] != nil || c["min"] != float64(0) || c["max"] != float64(40) || c["average_known"] != float64(20) {
		t.Fatal("ASSERT_USAGE_CONTEXT: fail")
	}
	t.Log("ASSERT_USAGE_CONTEXT: pass")
}
func TestTranscriptUsageSummaryBuckets(t *testing.T) {
	v := summaryValue(t, summaryFixture(t), 3)
	b, ok := v["buckets"].([]any)
	if !ok || len(b) != 2 {
		t.Fatal("ASSERT_USAGE_BUCKETS: fail")
	}
	a := b[0].(map[string]any)
	z := b[1].(map[string]any)
	if a["first_record"] != float64(1) || a["last_record"] != float64(3) || a["first_event_id"] != "1" || z["last_event_id"] != "4" || z["last_record"] != float64(4) || a["stats"].(map[string]any)["known_total_tokens"] != float64(47) {
		t.Fatal("ASSERT_USAGE_BUCKETS: fail")
	}
	t.Log("ASSERT_USAGE_BUCKETS: pass")
}
