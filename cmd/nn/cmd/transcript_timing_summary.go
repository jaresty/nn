package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"time"
)

// Timestamp parsing never replaces an invalid record clock with a message clock.
// Numeric native timestamps are integer Unix milliseconds, not seconds.
func ledgerTime(value any) (time.Time, string) {
	if value == nil {
		return time.Time{}, "missing"
	}
	var s string
	switch v := value.(type) {
	case string:
		s = v
	case json.RawMessage:
		if string(v) == "null" {
			return time.Time{}, "missing"
		}
		if json.Unmarshal(v, &s) != nil {
			var ms int64
			if json.Unmarshal(v, &ms) != nil {
				return time.Time{}, "invalid"
			}
			t := time.UnixMilli(ms).UTC()
			if t.Year() < 0 || t.Year() > 9999 {
				return time.Time{}, "invalid"
			}
			return t, "known"
		}
	default:
		return time.Time{}, "invalid"
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, "invalid"
	}
	return t, "known"
}

func ledgerEventError(e ledgerEvent) bool {
	if e["kind"] == "message" {
		m, _ := e["message"].(map[string]any)
		if m["role"] != "assistant" {
			return false
		}
		stop, _ := m["stop_reason"].(string)
		reason, _ := m["error_message"].(string)
		return stop == "error" || stop == "aborted" || reason != ""
	}
	if e["kind"] == "tool_result" {
		m, _ := e["tools"].(map[string]any)
		flag, _ := m["is_error"].(*bool)
		return flag != nil && *flag
	}
	return false
}

type timingStats struct {
	Status       string   `json:"status"`
	TotalSeconds *float64 `json:"total_seconds"`
	Intervals    int      `json:"intervals"`
	Unknown      int      `json:"unknown_intervals"`
	Negative     int      `json:"negative_intervals"`
	Seconds      float64  `json:"observed_seconds"`
	total        time.Duration
}

func (s *timingStats) finish() {
	s.Status = "unavailable"
	if s.Intervals > s.Negative {
		s.Status = "partial"
		if s.Unknown == 0 && s.Negative == 0 {
			s.Status = "complete"
			total := s.Seconds
			s.TotalSeconds = &total
		}
	}
}

type timingInterval struct {
	From                 string  `json:"from_event_id"`
	To                   string  `json:"to_event_id"`
	FromOrdinal          any     `json:"from_ordinal"`
	ToOrdinal            any     `json:"to_ordinal"`
	FromTimestamp        any     `json:"from_timestamp"`
	ToTimestamp          any     `json:"to_timestamp"`
	FromMessageTimestamp any     `json:"from_message_timestamp"`
	ToMessageTimestamp   any     `json:"to_message_timestamp"`
	Seconds              float64 `json:"seconds"`
	order                int
}

func (s *timingStats) add(from, to ledgerEvent, order int) (*timingInterval, error) {
	a, ak := ledgerTime(from["timestamp"])
	b, bk := ledgerTime(to["timestamp"])
	if ak != "known" || bk != "known" {
		s.Unknown++
		return nil, nil
	}
	d := b.Sub(a)
	if !a.Add(d).Equal(b) {
		return nil, fmt.Errorf("timing summary: interval exceeds duration range")
	}
	s.Intervals++
	if d < 0 {
		s.Negative++
		return nil, nil
	}
	if d > time.Duration(math.MaxInt64)-s.total {
		return nil, fmt.Errorf("timing summary: interval total overflow")
	}
	s.total += d
	s.Seconds = s.total.Seconds()
	am, _ := from["message"].(map[string]any)
	bm, _ := to["message"].(map[string]any)
	return &timingInterval{from["event_id"].(string), to["event_id"].(string), from["ordinal"], to["ordinal"], from["timestamp"], to["timestamp"], am["message_timestamp"], bm["message_timestamp"], d.Seconds(), order}, nil
}

func largestTiming(rows []timingInterval, limit int) []timingInterval {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Seconds != rows[j].Seconds {
			return rows[i].Seconds > rows[j].Seconds
		}
		return rows[i].order < rows[j].order
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows
}

func buildTimingSummary(session, id, schema, detail string, events []ledgerEvent, limit int, supplied string) ([]byte, error) {
	if limit < 0 || limit > 100 {
		return nil, fmt.Errorf("timing summary: --limit must be 0..100")
	}
	var gaps, tools timingStats
	clocks := map[string]int{"known": 0, "missing": 0, "invalid": 0}
	errors := map[string]int{"assistant_errors": 0, "assistant_aborted": 0, "assistant_error_messages": 0, "tool_errors": 0, "tool_error_status_unknown": 0}
	joins := map[string]int{"matched": 0, "missing": 0, "ambiguous": 0, "unavailable": 0}
	type transition struct {
		From  string      `json:"from_role"`
		To    string      `json:"to_role"`
		Stats timingStats `json:"stats"`
	}
	transitions := map[[2]string]*transition{}
	messages := []ledgerEvent{}
	byID := map[string]ledgerEvent{}
	for _, e := range events {
		byID[e["event_id"].(string)] = e
		if e["kind"] == "message" {
			messages = append(messages, e)
		}
	}
	ranked := []timingInterval{}
	rankedTools := []timingInterval{}
	for i, e := range messages {
		_, status := ledgerTime(e["timestamp"])
		clocks[status]++
		m, _ := e["message"].(map[string]any)
		if m["role"] == "assistant" {
			if m["stop_reason"] == "error" {
				errors["assistant_errors"]++
			}
			if m["stop_reason"] == "aborted" {
				errors["assistant_aborted"]++
			}
			if text, _ := m["error_message"].(string); text != "" {
				errors["assistant_error_messages"]++
			}
		}
		if i == 0 {
			continue
		}
		previous := messages[i-1]
		row, err := gaps.add(previous, e, i)
		if err != nil {
			return nil, err
		}
		if row != nil {
			ranked = append(ranked, *row)
		}
		pm, _ := previous["message"].(map[string]any)
		from, _ := pm["role"].(string)
		to, _ := m["role"].(string)
		key := [2]string{from, to}
		if transitions[key] == nil {
			transitions[key] = &transition{From: from, To: to}
		}
		if _, err := transitions[key].Stats.add(previous, e, i); err != nil {
			return nil, err
		}
	}
	for i, e := range events {
		if e["kind"] != "tool_result" {
			continue
		}
		m, _ := e["tools"].(map[string]any)
		flag, _ := m["is_error"].(*bool)
		if flag == nil {
			errors["tool_error_status_unknown"]++
		} else if *flag {
			errors["tool_errors"]++
		}
		match, _ := m["match_status"].(string)
		joins[match]++
		if match != "matched" {
			continue
		}
		target, _ := m["matched_event_id"].(string)
		call, ok := byID[target]
		if !ok {
			return nil, fmt.Errorf("timing summary: matched call unavailable")
		}
		// Extracted slots inherit their enclosing source-message clock. Retain the
		// message facet only for endpoint clock disclosure, never duplicate counting.
		from := cloneTimingEndpoint(call, byID)
		to := cloneTimingEndpoint(e, byID)
		row, err := tools.add(from, to, i)
		if err != nil {
			return nil, err
		}
		if row != nil {
			rankedTools = append(rankedTools, *row)
		}
	}
	gaps.finish()
	tools.finish()
	ordered := []*transition{}
	for _, v := range transitions {
		v.Stats.finish()
		ordered = append(ordered, v)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].From != ordered[j].From {
			return ordered[i].From < ordered[j].From
		}
		return ordered[i].To < ordered[j].To
	})
	var first, last any
	var elapsed any
	if len(messages) > 0 {
		first = timingBoundary(messages[0])
		last = timingBoundary(messages[len(messages)-1])
		var span timingStats
		row, err := span.add(messages[0], messages[len(messages)-1], 0)
		if err != nil {
			return nil, err
		}
		if row != nil {
			elapsed = row.Seconds
		}
	}
	gapCount, toolCount := len(ranked), len(rankedTools)
	ranked = largestTiming(ranked, limit)
	rankedTools = largestTiming(rankedTools, limit)
	ledger, err := buildLedgerPage(session, id, schema, detail, []string{"identity", "message", "tools"}, false, events, 1, "", "", true)
	if err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(session)
	if err != nil {
		return nil, err
	}
	result := map[string]any{
		"version": "nn.transcript.timing-summary/v1", "session": filepath.Clean(absolute), "agent_id": id, "schema": schema, "detail_status": detail, "ledger_snapshot": ledger.Snapshot, "limit": limit,
		"clock": "record_preferred_message_fallback", "message_records": len(messages), "timestamps": clocks, "first_message": first, "last_message": last, "elapsed_seconds": elapsed,
		"message_gaps": gaps, "transitions": ordered, "largest_gaps": ranked, "gaps_returned": len(ranked), "gaps_omitted": gapCount - len(ranked),
		"tool_intervals": tools, "tool_result_joins": joins, "largest_tool_intervals": rankedTools, "tool_intervals_returned": len(rankedTools), "tool_intervals_omitted": toolCount - len(rankedTools), "errors": errors,
		"interpretation": "Observed intervals in ledger order, not execution time, latency attribution, retry counts, or original-source completeness. Negative and unknown intervals are excluded from sums and rankings. Tool intervals may overlap; do not add them to message gaps.",
		"retry_evidence": "unavailable: no normalized explicit retry/backoff records",
	}
	body, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(append([]byte("nn transcript timing summary v1\x00"), body...))
	snapshot := hex.EncodeToString(h[:])
	if supplied != "" && supplied != snapshot {
		return nil, fmt.Errorf("timing summary: stale or mismatched --snapshot")
	}
	result["snapshot"] = snapshot
	body, err = json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if len(body)+1 > graphBodiesPageMaxBytes {
		return nil, fmt.Errorf("timing summary exceeds 48000 bytes; reduce --limit (transition groups are never silently omitted)")
	}
	return append(body, '\n'), nil
}

func cloneTimingEndpoint(e ledgerEvent, byID map[string]ledgerEvent) ledgerEvent {
	out := ledgerEvent{}
	for k, v := range e {
		out[k] = v
	}
	parent, _ := e["message_event_id"].(string)
	if m := byID[parent]; m != nil {
		out["message"] = m["message"]
	}
	return out
}

func timingBoundary(e ledgerEvent) map[string]any {
	m, _ := e["message"].(map[string]any)
	return map[string]any{"event_id": e["event_id"], "ordinal": e["ordinal"], "timestamp": e["timestamp"], "timestamp_source": e["timestamp_source"], "record_timestamp": m["record_timestamp"], "message_timestamp": m["message_timestamp"]}
}
