package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jaresty/nn/internal/attention"
)

type attentionEvidence struct {
	EventID string `json:"event_id"`
	Path    string `json:"path"`
	Ordinal int    `json:"record_ordinal"`
	Role    string `json:"role"`
	Summary string `json:"summary"`
}

type attentionWindow struct {
	Requested         int    `json:"requested_work_events"`
	Selected          int    `json:"selected_work_events"`
	Earlier           int    `json:"earlier_ledger_records_omitted"`
	UnknownTimestamps int    `json:"unknown_timestamps"`
	FirstEvent        string `json:"first_event_id,omitempty"`
	LastEvent         string `json:"last_event_id,omitempty"`
}

// A work event is one owned assistant or tool-result message, not its ledger
// facets. Select in canonical ledger order, then count unique invocation IDs.
func collectAttention(records []ledgerRecord, status, id string, n int) (attention.Metrics, attentionWindow, []attentionEvidence, error) {
	metrics := attention.Metrics{Available: status == "available"}
	window := attentionWindow{Requested: n}
	indices := []int{}
	for i := len(records) - 1; i >= 0; i-- {
		lr := records[i]
		if len(lr.Record.Message) > 1024*1024 {
			return metrics, window, nil, fmt.Errorf("attention: selected candidate message exceeds 1 MiB")
		}
		if lr.Lifecycle {
			continue
		}
		_, role := attentionRecordRole(lr)
		if role != "assistant" && role != "toolResult" && role != "tool" && role != "unknown" {
			continue
		}
		indices = append(indices, i)
		if len(indices) == n {
			window.Earlier = i
			break
		}
	}
	evidence := []attentionEvidence{}
	seen := map[string]string{}
	rejected := attentionRejectedCommands(records, indices, id)
	operations := 0
	for j := len(indices) - 1; j >= 0; j-- {
		lr := records[indices[j]]
		m, role := attentionRecordRole(lr)
		if role == "unknown" || m == nil {
			metrics.Unknown++
		}
		eventID := ledgerID(lr.Path, lr.Record.RecordOrdinal, id, "message")
		if _, ok := reviewTimestamp(lr.Record, m); !ok {
			window.UnknownTimestamps++
		}
		summaries := []string{}
		if role == "assistant" {
			var blocks []map[string]json.RawMessage
			if err := json.Unmarshal(m["content"], &blocks); err != nil || blocks == nil {
				var text string
				if json.Unmarshal(m["content"], &text) != nil || string(m["content"]) == "null" || len(m["content"]) == 0 {
					metrics.Unknown++
				} else {
					summaries = append(summaries, text)
				}
			}
			for _, b := range blocks {
				typ := ledgerString(b, "type")
				if typ == "text" {
					summaries = append(summaries, ledgerString(b, "text"))
					continue
				}
				if typ == "thinking" || typ == "reasoning" {
					continue
				}
				if typ != "toolCall" && typ != "tool_use" {
					metrics.Unknown++
					continue
				}
				operations++
				if operations > 2000 {
					return metrics, window, nil, fmt.Errorf("attention: operation limit exceeded")
				}
				name, callID := ledgerString(b, "name"), ledgerString(b, "id")
				args := b["arguments"]
				if len(args) == 0 {
					args = b["input"]
				}
				var a map[string]json.RawMessage
				valid := json.Unmarshal(args, &a) == nil && a != nil
				class := attentionToolClass(name)
				if !valid || callID == "" {
					class = "unknown"
				}
				canonical, _ := json.Marshal(a)
				fingerprint := name + "\x00" + string(canonical)
				if callID != "" {
					if prior, ok := seen[callID]; ok {
						if prior == fingerprint {
							metrics.Duplicates++
							summaries = append(summaries, "duplicate invocation "+callID)
							continue
						}
						metrics.Unknown++
						summaries = append(summaries, "ambiguous invocation ID "+callID)
						continue
					}
					seen[callID] = fingerprint
				}
				if resultID, ok := rejected[callID]; ok {
					metrics.Rejected++
					summaries = append(summaries, name+" [validation rejected before execution]; result "+resultID)
					continue
				}
				switch class {
				case "command":
					command := ledgerString(a, "command")
					if command == "" {
						metrics.Unknown++
						class = "unknown"
					} else {
						metrics.Commands++
					}
					summaries = append(summaries, name+" ["+class+"]: "+command)
				case "edit":
					path := ledgerString(a, "path")
					if path == "" {
						path = ledgerString(a, "file_path")
					}
					if path == "" {
						metrics.Unknown++
					} else {
						metrics.Edits++
					}
					summaries = append(summaries, name+" [recognized edit invocation]: "+path)
				case "neutral":
					metrics.Neutral++
					summaries = append(summaries, name+" [neutral invocation]")
				default:
					metrics.Unknown++
					summaries = append(summaries, name+" [unknown classification]")
				}
			}
		} else {
			// Results consume work-window slots but never add invocations or successes.
			summaries = append(summaries, "tool result (not counted as another operation)")
		}
		summary := cleanBundleText(strings.Join(summaries, "; "))
		runes := []rune(summary)
		if len(runes) > 320 {
			summary = string(runes[:320]) + "… [excerpt truncated]"
		}
		evidence = append(evidence, attentionEvidence{EventID: eventID, Path: lr.Path, Ordinal: lr.Record.RecordOrdinal, Role: role, Summary: summary})
	}
	window.Selected = len(evidence)
	if window.Selected > 0 {
		window.FirstEvent = evidence[0].EventID
		window.LastEvent = evidence[len(evidence)-1].EventID
	}
	return metrics, window, evidence, nil
}

// Claude serializes tool results as blocks in user messages; those are work,
// while ordinary user instructions are not. Unknown message shapes fail closed.
func attentionRecordRole(lr ledgerRecord) (map[string]json.RawMessage, string) {
	var m map[string]json.RawMessage
	if json.Unmarshal(lr.Record.Message, &m) != nil || m == nil {
		return m, "unknown"
	}
	role := ledgerString(m, "role")
	if role == "" {
		role = lr.Record.Type
	}
	if role == "user" {
		var blocks []map[string]json.RawMessage
		if json.Unmarshal(m["content"], &blocks) == nil {
			for _, b := range blocks {
				if ledgerString(b, "type") == "tool_result" {
					return m, "toolResult"
				}
			}
		}
	}
	if role == "message" || role == "" {
		role = "unknown"
	}
	return m, role
}

// Exact vocabulary only. No shell parsing and no fuzzy tool-name matching.
// Shell invocation classification is known; its filesystem side effects are not.
func attentionToolClass(name string) string {
	switch name {
	case "bash", "Bash", "functions.bash":
		return "command"
	case "edit", "Edit", "write", "Write", "functions.edit", "functions.write":
		return "edit"
	case "read", "Read", "grep", "Grep", "find", "Glob", "ls", "functions.read":
		return "neutral"
	}
	return "unknown"
}
