package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

type toolVolume struct {
	KnownTotal int64  `json:"known_total"`
	Known      int    `json:"known_records"`
	Unknown    int    `json:"unknown_records"`
	Total      *int64 `json:"total"`
	Status     string `json:"status"`
}

func (v *toolVolume) add(n *int64) error {
	if n == nil {
		v.Unknown++
	} else {
		if *n < 0 || *n > math.MaxInt64-v.KnownTotal {
			return fmt.Errorf("tool summary: negative size or integer overflow")
		}
		v.KnownTotal += *n
		v.Known++
	}
	v.Status = "unavailable"
	v.Total = nil
	if v.Known > 0 {
		v.Status = "partial"
		if v.Unknown == 0 {
			v.Status = "complete"
			value := v.KnownTotal
			v.Total = &value
		}
	}
	return nil
}

type toolResultSize struct {
	Characters *int64 `json:"text_characters"`
	Bytes      *int64 `json:"text_bytes"`
	Content    *int64 `json:"content_bytes"`
}
type toolSummaryFields struct {
	Name      string         `json:"name"`
	CallID    string         `json:"call_id"`
	Match     string         `json:"match_status"`
	Matched   *string        `json:"matched_event_id"`
	Arguments *int64         `json:"arguments_bytes"`
	Size      toolResultSize `json:"result_size"`
}
type toolSummaryRow struct {
	ID, Kind string
	Ordinal  int
	Fields   toolSummaryFields
	Event    ledgerEvent
}
type toolSummaryStats struct {
	Calls       int                    `json:"calls"`
	Results     int                    `json:"results"`
	CallJoins   map[string]int         `json:"call_joins"`
	ResultJoins map[string]int         `json:"result_joins"`
	Sizes       map[string]*toolVolume `json:"sizes"`
}

func newToolSummaryStats() *toolSummaryStats {
	joins := func() map[string]int {
		return map[string]int{"matched": 0, "missing": 0, "ambiguous": 0, "unavailable": 0}
	}
	s := &toolSummaryStats{CallJoins: joins(), ResultJoins: joins(), Sizes: map[string]*toolVolume{}}
	for _, k := range []string{"argument_bytes", "result_text_characters", "result_text_bytes", "result_content_bytes"} {
		s.Sizes[k] = &toolVolume{Status: "unavailable"}
	}
	return s
}
func (s *toolSummaryStats) add(r toolSummaryRow) error {
	if r.Kind == "tool_call" {
		s.Calls++
		s.CallJoins[r.Fields.Match]++
		return s.Sizes["argument_bytes"].add(r.Fields.Arguments)
	}
	s.Results++
	s.ResultJoins[r.Fields.Match]++
	for k, n := range map[string]*int64{"result_text_characters": r.Fields.Size.Characters, "result_text_bytes": r.Fields.Size.Bytes, "result_content_bytes": r.Fields.Size.Content} {
		if err := s.Sizes[k].add(n); err != nil {
			return err
		}
	}
	return nil
}
func toolName(name string) any {
	if name == "" {
		return nil
	}
	return name
}
func toolPreview(s string) (string, bool) {
	s = strings.ToValidUTF8(s, "\ufffd")
	if len(s) <= 512 {
		return s, false
	}
	n := 512
	for !utf8.ValidString(s[:n]) {
		n--
	}
	return s[:n], true
}
func toolCallView(r toolSummaryRow) (map[string]any, error) {
	result := map[string]any{"event_id": r.ID, "tool": toolName(r.Fields.Name), "arguments_bytes": r.Fields.Arguments, "command": nil, "command_truncated": false, "arguments_preview": nil, "arguments_truncated": false, "arguments_sha256": nil}
	raw, ok := r.Event["payload"].(json.RawMessage)
	if !ok && r.Event["payload"] != nil {
		var err error
		raw, err = json.Marshal(r.Event["payload"])
		if err != nil {
			return nil, err
		}
	}
	var p struct {
		Arguments json.RawMessage `json:"arguments"`
		Input     json.RawMessage `json:"input"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
	}
	args := p.Arguments
	if len(args) == 0 {
		args = p.Input
	}
	if len(args) == 0 || string(args) == "null" {
		return result, nil
	}
	preview, truncated := toolPreview(string(args))
	result["arguments_preview"] = preview
	result["arguments_truncated"] = truncated
	hash := sha256.Sum256(args)
	result["arguments_sha256"] = hex.EncodeToString(hash[:])
	var object map[string]json.RawMessage
	if json.Unmarshal(args, &object) == nil {
		var command *string
		if json.Unmarshal(object["command"], &command) == nil && command != nil {
			p, t := toolPreview(*command)
			result["command"] = p
			result["command_truncated"] = t
		}
	}
	return result, nil
}

func buildToolSummary(session, id, schema, detail string, events []ledgerEvent, limit int, groupBy, supplied string) ([]byte, error) {
	if limit < 0 || limit > 100 {
		return nil, fmt.Errorf("tool summary: --limit must be between 0 and 100")
	}
	if groupBy != "" && groupBy != "tool" {
		return nil, fmt.Errorf("tool summary: --group-by must be tool")
	}
	rows := []toolSummaryRow{}
	byID := map[string]toolSummaryRow{}
	for _, e := range events {
		kind, _ := e["kind"].(string)
		if kind != "tool_call" && kind != "tool_result" {
			continue
		}
		id, _ := e["event_id"].(string)
		if _, exists := byID[id]; id == "" || exists {
			return nil, fmt.Errorf("tool summary: missing or duplicate event identity")
		}
		b, err := json.Marshal(e["tools"])
		if err != nil {
			return nil, err
		}
		var fields toolSummaryFields
		if err = json.Unmarshal(b, &fields); err != nil {
			return nil, err
		}
		switch fields.Match {
		case "matched", "missing", "ambiguous", "unavailable":
		default:
			return nil, fmt.Errorf("tool summary: invalid join status")
		}
		ordinal, ok := e["ordinal"].(int)
		if !ok {
			return nil, fmt.Errorf("tool summary: invalid ledger ordinal")
		}
		r := toolSummaryRow{id, kind, ordinal, fields, e}
		rows = append(rows, r)
		byID[id] = r
	}
	stats := newToolSummaryStats()
	groupStats := map[string]*toolSummaryStats{}
	results := []toolSummaryRow{}
	for _, r := range rows {
		name := r.Fields.Name
		if r.Fields.Match == "matched" {
			if r.Fields.Matched == nil {
				return nil, fmt.Errorf("tool summary: inconsistent matched join")
			}
			other, ok := byID[*r.Fields.Matched]
			if !ok || other.Kind == r.Kind || other.Fields.Match != "matched" || other.Fields.Matched == nil || *other.Fields.Matched != r.ID || r.Fields.CallID == "" || r.Fields.CallID != other.Fields.CallID {
				return nil, fmt.Errorf("tool summary: inconsistent matched join")
			}
			if r.Kind == "tool_result" && name == "" {
				name = other.Fields.Name
			}
		}
		if err := stats.add(r); err != nil {
			return nil, err
		}
		if groupBy == "tool" {
			if groupStats[name] == nil {
				groupStats[name] = newToolSummaryStats()
			}
			if err := groupStats[name].add(r); err != nil {
				return nil, err
			}
		}
		if r.Kind == "tool_result" {
			r.Fields.Name = name
			results = append(results, r)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		a, b := results[i], results[j]
		x, y := a.Fields.Size.Characters, b.Fields.Size.Characters
		if (x == nil) != (y == nil) {
			return x != nil
		}
		if x != nil && *x != *y {
			return *x > *y
		}
		if a.Ordinal != b.Ordinal {
			return a.Ordinal < b.Ordinal
		}
		return a.ID < b.ID
	})
	count := len(results)
	if count > limit {
		count = limit
	}
	largest := []map[string]any{}
	for _, r := range results[:count] {
		var call any
		if r.Fields.Match == "matched" {
			view, err := toolCallView(byID[*r.Fields.Matched])
			if err != nil {
				return nil, err
			}
			call = view
		}
		largest = append(largest, map[string]any{"event_id": r.ID, "ordinal": r.Ordinal, "tool": toolName(r.Fields.Name), "match_status": r.Fields.Match, "result_size": r.Fields.Size, "call": call})
	}
	groups := []map[string]any{}
	names := []string{}
	for name := range groupStats {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		groups = append(groups, map[string]any{"tool": toolName(name), "stats": groupStats[name]})
	}
	// Metadata ledger snapshot excludes payloads; displayed argument hashes are bound below.
	metadata := make([]ledgerEvent, len(events))
	for i, e := range events {
		m := ledgerEvent{}
		for k, v := range e {
			if k != "payload" {
				m[k] = v
			}
		}
		metadata[i] = m
	}
	ledger, err := buildLedgerPage(session, id, schema, detail, []string{"identity", "tools"}, false, metadata, 1, "", "", true)
	if err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(session)
	if err != nil {
		return nil, err
	}
	result := map[string]any{"version": "nn.transcript.tool-summary/v1", "session": filepath.Clean(absolute), "agent_id": id, "schema": schema, "detail_status": detail, "ledger_snapshot": ledger.Snapshot, "limit": limit, "group_by": groupBy, "ranking": "result_text_characters_desc_known_first", "stats": stats, "largest_results": largest, "results_returned": count, "results_omitted": len(results) - count, "groups": groups}
	b, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(append([]byte("nn transcript tool summary v1\x00"), b...))
	snapshot := hex.EncodeToString(h[:])
	if supplied != "" && supplied != snapshot {
		return nil, fmt.Errorf("tool summary: stale or mismatched --snapshot")
	}
	result["snapshot"] = snapshot
	b, err = json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if len(b)+1 > graphBodiesPageMaxBytes {
		return nil, fmt.Errorf("tool summary exceeds 48000 bytes; reduce --limit (0 omits results) or omit --group-by")
	}
	return append(b, '\n'), nil
}
