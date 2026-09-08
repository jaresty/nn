package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"unicode/utf8"
)

type ledgerEvent map[string]any

func ledgerID(path string, record int, owner, slot string) string {
	b, _ := json.Marshal([]any{path, record, owner, slot})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func ledgerString(m map[string]json.RawMessage, key string) string {
	var s string
	_ = json.Unmarshal(m[key], &s)
	return s
}

// First well-typed, non-null native field wins, including an explicit empty string.
func ledgerNativeString(fields map[string]json.RawMessage, keys ...string) any {
	for _, key := range keys {
		var value *string
		if json.Unmarshal(fields[key], &value) == nil && value != nil {
			return *value
		}
	}
	return nil
}

func ledgerSize(raw json.RawMessage) map[string]any {
	if len(raw) == 0 || string(raw) == "null" {
		return map[string]any{"content_bytes": nil, "text_bytes": nil, "text_characters": nil}
	}
	text := ledgerText(raw)
	return map[string]any{"content_bytes": len(raw), "text_bytes": len(text), "text_characters": utf8.RuneCountInString(text)}
}

// Text sizing measures decoded textual content only, not arguments, opaque media,
// reasoning signatures or tokenizer input. No separators are synthesized.
func ledgerText(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	var blocks []map[string]json.RawMessage
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}
	for _, b := range blocks {
		switch ledgerString(b, "type") {
		case "text":
			text += ledgerString(b, "text")
		case "tool_result":
			text += ledgerText(b["content"])
		}
	}
	return text
}

func ledgerUsage(raw json.RawMessage) (map[string]any, error) {
	var m map[string]json.RawMessage
	if len(raw) > 0 && json.Unmarshal(raw, &m) != nil {
		return nil, fmt.Errorf("events: malformed usage object")
	}
	result := map[string]any{"status": "unavailable", "scope": "source_message", "total_tokens": nil, "context_tokens": nil}
	fields := []struct{ name, a, b string }{{"input_tokens", "input", "input_tokens"}, {"output_tokens", "output", "output_tokens"}, {"cache_read_tokens", "cacheRead", "cache_read_input_tokens"}, {"cache_creation_tokens", "cacheWrite", "cache_creation_input_tokens"}}
	known := 0
	var total int64
	for _, f := range fields {
		var value *int64
		for _, k := range []string{f.a, f.b} {
			raw, ok := m[k]
			if !ok || string(raw) == "null" {
				continue
			}
			var n int64
			if json.Unmarshal(raw, &n) != nil || n < 0 {
				return nil, fmt.Errorf("events: invalid usage counter %s", k)
			}
			if value == nil {
				value = new(int64)
			}
			if n > math.MaxInt64-*value {
				return nil, fmt.Errorf("events: usage overflow")
			}
			*value += n
		}
		result[f.name] = value
		if value != nil {
			known++
			if *value > math.MaxInt64-total {
				return nil, fmt.Errorf("events: usage overflow")
			}
			total += *value
		}
	}
	result["known_total_tokens"] = total
	if known > 0 {
		result["status"] = "partial"
	}
	if known == 4 {
		result["status"] = "complete"
		result["total_tokens"] = total
	}
	input := result["input_tokens"].(*int64)
	cache := result["cache_read_tokens"].(*int64)
	if input != nil && cache != nil {
		result["context_tokens"] = *input + *cache
	}
	return result, nil
}

func projectLedger(records []ledgerRecord, owner string, selects []string, payload bool) ([]ledgerEvent, error) {
	events := []ledgerEvent{}
	for _, src := range records {
		r := src.Record
		nativeID := r.ID
		if nativeID == "" {
			nativeID = r.UUID
		}
		base := func(kind, slot string, msg map[string]json.RawMessage) ledgerEvent {
			var timestamp any
			origin := "unavailable"
			if r.Timestamp != "" {
				timestamp = r.Timestamp
				origin = "record"
			} else if raw := msg["timestamp"]; len(raw) > 0 && string(raw) != "null" {
				var v any
				if json.Unmarshal(raw, &v) == nil {
					switch v.(type) {
					case string, float64:
						timestamp = json.RawMessage(raw)
						origin = "message"
					}
				}
			}
			return ledgerEvent{"event_id": ledgerID(src.Path, r.RecordOrdinal, owner, slot), "agent_id": owner, "ordinal": len(events) + 1, "kind": kind, "timestamp": timestamp, "timestamp_source": origin, "source": map[string]any{"path": src.Path, "record_ordinal": r.RecordOrdinal, "record_id": nativeID}}
		}
		if src.Lifecycle {
			var data map[string]json.RawMessage
			if json.Unmarshal(r.Data, &data) != nil {
				continue
			}
			e := base("lifecycle", "lifecycle", nil)
			var start, end *int64
			_ = json.Unmarshal(data["startedAt"], &start)
			_ = json.Unmarshal(data["completedAt"], &end)
			e["lifecycle"] = map[string]any{"status": ledgerString(data, "status"), "started_at": start, "completed_at": end, "scope": "producer_terminal_record"}
			if payload {
				e["payload"] = r.Data
			}
			events = append(events, e)
			continue
		}
		var msg map[string]json.RawMessage
		if json.Unmarshal(r.Message, &msg) != nil || msg == nil {
			continue
		}
		role := ledgerString(msg, "role")
		if role == "" {
			role = r.Type
		}
		e := base("message", "message", msg)
		size := ledgerSize(msg["content"])
		size["role"] = role
		size["model"] = ledgerString(msg, "model")
		size["record_timestamp"] = nil
		if r.Timestamp != "" {
			size["record_timestamp"] = r.Timestamp
		}
		size["message_timestamp"] = nil
		if raw := msg["timestamp"]; len(raw) > 0 {
			size["message_timestamp"] = raw
		}
		size["stop_reason"] = ledgerNativeString(msg, "stopReason", "stop_reason")
		size["error_message"] = ledgerNativeString(msg, "errorMessage", "error_message")
		e["message"] = size
		if role == "assistant" && r.Type != "toolResult" {
			u, err := ledgerUsage(msg["usage"])
			if err != nil {
				return nil, err
			}
			e["usage"] = u
		}
		if payload {
			e["payload"] = r.Message
		}
		events = append(events, e)
		addTool := func(kind, slot string, b map[string]json.RawMessage, raw json.RawMessage) {
			te := base(kind, slot, msg)
			te["message_event_id"] = e["event_id"]
			tool := map[string]any{"call_id": "", "name": "", "match_status": "unavailable", "matched_event_id": nil, "is_error": nil, "arguments_bytes": nil}
			if kind == "tool_call" {
				tool["call_id"] = ledgerString(b, "id")
				tool["name"] = ledgerString(b, "name")
				args := b["arguments"]
				if len(args) == 0 {
					args = b["input"]
				}
				if len(args) > 0 && string(args) != "null" {
					tool["arguments_bytes"] = len(args)
				}
			} else {
				id := ledgerString(b, "toolCallId")
				if id == "" {
					id = ledgerString(b, "tool_use_id")
				}
				tool["call_id"] = id
				tool["name"] = ledgerString(b, "toolName")
				errorRaw := b["isError"]
				if len(errorRaw) == 0 {
					errorRaw = b["is_error"]
				}
				var flag *bool
				_ = json.Unmarshal(errorRaw, &flag)
				tool["is_error"] = flag
				tool["result_size"] = ledgerSize(b["content"])
			}
			te["tools"] = tool
			if payload {
				te["payload"] = raw
			}
			events = append(events, te)
		}
		if role == "toolResult" || role == "tool_result" || r.Type == "toolResult" {
			addTool("tool_result", "result", msg, r.Message)
			continue
		}
		var blocks []json.RawMessage
		_ = json.Unmarshal(msg["content"], &blocks)
		for i, raw := range blocks {
			var b map[string]json.RawMessage
			if json.Unmarshal(raw, &b) != nil {
				continue
			}
			switch ledgerString(b, "type") {
			case "toolCall", "tool_use":
				addTool("tool_call", fmt.Sprintf("block:%d", i), b, raw)
			case "tool_result":
				addTool("tool_result", fmt.Sprintf("block:%d", i), b, raw)
			}
		}
	}
	// Resolve only unique one-to-one source IDs. Missing and duplicate IDs remain explicit.
	calls, results := map[string][]ledgerEvent{}, map[string][]ledgerEvent{}
	for _, e := range events {
		if t, ok := e["tools"].(map[string]any); ok {
			id := t["call_id"].(string)
			if id != "" {
				if e["kind"] == "tool_call" {
					calls[id] = append(calls[id], e)
				} else {
					results[id] = append(results[id], e)
				}
			}
		}
	}
	for _, e := range events {
		if t, ok := e["tools"].(map[string]any); ok {
			id := t["call_id"].(string)
			if id != "" {
				c, r := calls[id], results[id]
				switch {
				case len(c) > 1 || len(r) > 1:
					t["match_status"] = "ambiguous"
				case len(c) == 1 && len(r) == 1:
					t["match_status"] = "matched"
					target := c[0]
					if e["kind"] == "tool_call" {
						target = r[0]
					}
					t["matched_event_id"] = target["event_id"]
				default:
					t["match_status"] = "missing"
				}
			}
		}
	}
	keep := map[string]bool{}
	for _, s := range selects {
		keep[s] = true
	}
	for _, e := range events {
		for _, facet := range []string{"message", "usage", "tools", "lifecycle"} {
			if !keep[facet] {
				delete(e, facet)
			}
		}
	}
	return events, nil
}
