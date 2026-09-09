package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
)

// Narrow Pi producer-format adapter. Call/result identity and owned source are
// mandatory; prose adjacency, execution errors, and ambiguous joins never suffice.
func attentionRejectedCommands(records []ledgerRecord, indices []int, agent string) map[string]string {
	type call struct {
		index int
		name  string
		args  map[string]json.RawMessage
	}
	calls := map[string][]call{}
	results := map[string][]int{}
	for _, i := range indices {
		m, role := attentionRecordRole(records[i])
		if role == "toolResult" {
			if id := ledgerString(m, "toolCallId"); id != "" {
				results[id] = append(results[id], i)
			}
		}
		if role != "assistant" {
			continue
		}
		var blocks []map[string]json.RawMessage
		if json.Unmarshal(m["content"], &blocks) != nil {
			continue
		}
		for _, b := range blocks {
			typ := ledgerString(b, "type")
			if typ != "toolCall" && typ != "tool_use" {
				continue
			}
			id := ledgerString(b, "id")
			if id == "" {
				continue
			}
			var args map[string]json.RawMessage
			// Only Pi toolCall arguments can qualify. Other forms still make IDs ambiguous.
			if typ == "toolCall" {
				_ = json.Unmarshal(b["arguments"], &args)
			}
			calls[id] = append(calls[id], call{i, ledgerString(b, "name"), args})
		}
	}
	rejected := map[string]string{}
	for id, cs := range calls {
		rs := results[id]
		if len(cs) != 1 || len(rs) != 1 {
			continue
		}
		c := cs[0]
		ri := rs[0]
		if c.name != "bash" || c.args == nil {
			continue
		}
		if _, exists := c.args["command"]; exists {
			continue
		}
		before, after := records[c.index], records[ri]
		if ri <= c.index || before.Path != after.Path || after.Record.RecordOrdinal <= before.Record.RecordOrdinal {
			continue
		}
		m, _ := attentionRecordRole(after)
		if ledgerString(m, "role") != "toolResult" || ledgerString(m, "toolName") != c.name || string(m["isError"]) != "true" {
			continue
		}
		var blocks []map[string]json.RawMessage
		if json.Unmarshal(m["content"], &blocks) != nil || len(blocks) != 1 || ledgerString(blocks[0], "type") != "text" {
			continue
		}
		text := ledgerString(blocks[0], "text")
		const prefix = "Validation failed for tool \"bash\":\n"
		const separator = "\n\nReceived arguments:\n"
		if !strings.HasPrefix(text, prefix) {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(text, prefix), separator, 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) != "- command: must have required properties command" {
			continue
		}
		var received map[string]json.RawMessage
		if json.Unmarshal([]byte(parts[1]), &received) != nil || received == nil {
			continue
		}
		want, _ := json.Marshal(c.args)
		got, _ := json.Marshal(received)
		if !bytes.Equal(want, got) {
			continue
		}
		rejected[id] = ledgerID(after.Path, after.Record.RecordOrdinal, agent, "message")
	}
	return rejected
}
