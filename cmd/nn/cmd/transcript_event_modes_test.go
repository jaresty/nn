package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranscriptPiRawPreservesMessagesInEveryRoute(t *testing.T) {
	for _, route := range []string{"root", "inline", "direct", "resolved"} {
		t.Run(route, func(t *testing.T) {
			dir := t.TempDir()
			session := filepath.Join(dir, "session.jsonl")
			agent, owner := "AAA", `,"agentId":"AAA","isSidechain":true`
			if route == "root" {
				agent, owner = "ROOT", ""
			}
			events := `{"type":"message"` + owner + `,"message":{"role":"assistant","content":"VISIBLE_TEXT","usage":{"input":123,"output":7},"rawOnly":"RAW_MARKER"}}` + "\n" +
				`{"type":"message"` + owner + `,"message":{"role":"toolResult","content":[{"type":"text","text":"RESULT_MARKER"}]}}` + "\n"
			switch route {
			case "root", "inline":
				writeTranscriptFile(t, session, `{"type":"session"}`+"\n"+events)
			case "direct":
				session = filepath.Join(dir, "direct.output")
				writeTranscriptFile(t, session, events)
			case "resolved":
				side := filepath.Join(dir, "pi-subagents-test", "session", "tasks", "AAA.output")
				writeTranscriptFile(t, side, events)
				writeTranscriptFile(t, session, `{"type":"session"}`+"\n"+
					`{"type":"message","message":{"role":"toolResult","details":{"agentId":"AAA","status":"background","fullOutputPath":`+mustJSONString(t, side)+`}}}`+"\n")
			}
			_, execute := setupNotebook(t)
			meaningful, _ := reconstructTranscriptProjection(t, execute, session, agent, false)
			raw, _ := reconstructTranscriptProjection(t, execute, session, agent, true)
			if raw == meaningful || !strings.Contains(raw, "RAW_MARKER") || !strings.Contains(raw, `"usage"`) || !strings.Contains(raw, "RESULT_MARKER") {
				t.Errorf("ASSERT_PI_RAW_PRESERVES_MESSAGE_FIELDS: route=%s raw=%q", route, raw)
			}
			if !strings.Contains(meaningful, "VISIBLE_TEXT") || strings.Contains(meaningful, "RAW_MARKER") || strings.Contains(meaningful, "RESULT_MARKER") {
				t.Errorf("ASSERT_PI_MEANINGFUL_OMITS_TOOL_RESULTS: route=%s meaningful=%q", route, meaningful)
			}
		})
	}
}

func TestTranscriptSearchShowMeaningfulParity(t *testing.T) {
	for _, role := range []string{"assistant", "user", "toolResult", "tool_result"} {
		t.Run(role, func(t *testing.T) {
			session := filepath.Join(t.TempDir(), "pi.jsonl")
			writeTranscriptFile(t, session, `{"type":"session"}`+"\n"+
				`{"type":"message","message":{"role":"`+role+`","content":[{"type":"text","text":"PARITY_MARKER"}]}}`+"\n")
			_, execute := setupNotebook(t)
			show, err := execute("transcript", "show", session, "ROOT")
			if err != nil {
				t.Fatal(err)
			}
			search, err := execute("transcript", "search", "PARITY_MARKER", "--session", session, "--json")
			if err != nil {
				t.Fatal(err)
			}
			var result struct {
				Matches []json.RawMessage `json:"matches"`
			}
			if err := json.Unmarshal([]byte(search), &result); err != nil {
				t.Fatal(err)
			}
			wantVisible := role == "assistant" || role == "user"
			if strings.Contains(show, "PARITY_MARKER") != wantVisible || (len(result.Matches) == 1) != wantVisible {
				t.Fatalf("ASSERT_SEARCH_SHOW_MEANINGFUL_PARITY: role=%s visible=%v matches=%d", role, strings.Contains(show, "PARITY_MARKER"), len(result.Matches))
			}
		})
	}
}
