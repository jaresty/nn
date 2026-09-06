package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func nativeToolResultFixture(t *testing.T, mixed bool) (string, string) {
	t.Helper()
	dir := t.TempDir()
	side := filepath.Join(dir, "pi-subagents-test", "session", "tasks", "AAA.output")
	message := `{"role":"toolResult","toolName":"Bash","toolCallId":"call-native","isError":true,"nativeField":"kept","usage":{"input":999},"content":[{"type":"text","text":` +
		mustJSONString(t, "OWNED_NATIVE_RESULT "+strings.Repeat("α<&😀", 9000)) + `}]}`
	events := `{"type":"toolResult","isSidechain":true,"agentId":"AAA","message":` + message + `}` + "\n" +
		`{"type":"toolResult","isSidechain":true,"agentId":"BBB","message":{"role":"toolResult","content":"FOREIGN_NATIVE_RESULT"}}` + "\n" +
		`{"type":"toolResult","isSidechain":true,"message":{"role":"toolResult","content":"UNOWNED_NATIVE_RESULT"}}` + "\n"
	if mixed {
		events = `{"type":"assistant","isSidechain":true,"agentId":"AAA","message":{"role":"assistant","content":"VISIBLE_ASSISTANT","usage":{"input":3,"output":4}}}` + "\n" + events
	}
	writeTranscriptFile(t, side, events)
	parent := filepath.Join(dir, "parent.jsonl")
	writeTranscriptFile(t, parent, `{"type":"session"}`+"\n"+
		`{"type":"message","message":{"role":"toolResult","details":{"agentId":"AAA","status":"background","fullOutputPath":`+mustJSONString(t, side)+`}}}`+"\n")
	return parent, side
}

func TestTranscriptNativeToolResultRawShowAndSearch(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		parent, side := nativeToolResultFixture(t, mixed)
		for route, source := range map[string]string{"direct": side, "resolved": parent} {
			t.Run(route+map[bool]string{false: "-only", true: "-mixed"}[mixed], func(t *testing.T) {
				_, execute := setupNotebook(t)
				raw, _ := reconstructTranscriptProjection(t, execute, source, "AAA", true)
				if strings.Count(raw, "OWNED_NATIVE_RESULT") != 1 || !strings.Contains(raw, `"nativeField":"kept"`) || !strings.Contains(raw, `"isError":true`) {
					t.Errorf("ASSERT_NATIVE_PI_RESULTS_RAW_SHOW: missing native payload/fields in %s", route)
				}
				if strings.Contains(raw, "FOREIGN_NATIVE_RESULT") || strings.Contains(raw, "UNOWNED_NATIVE_RESULT") {
					t.Errorf("ASSERT_NATIVE_PI_RESULTS_RAW_SHOW: foreign or unowned payload leaked")
				}
				meaningful, _ := reconstructTranscriptProjection(t, execute, source, "AAA", false)
				if strings.Contains(meaningful, "NATIVE_RESULT") {
					t.Error("native tool-result payload leaked into meaningful show")
				}
				if mixed && !strings.Contains(meaningful, "VISIBLE_ASSISTANT") {
					t.Error("meaningful assistant event disappeared")
				}
			})
		}
		t.Run(map[bool]string{false: "search-only", true: "search-mixed"}[mixed], func(t *testing.T) {
			_, execute := setupNotebook(t)
			for _, raw := range []bool{false, true} {
				args := []string{"transcript", "search", "OWNED_NATIVE_RESULT", "--session", side, "--agent", "AAA", "--json"}
				if raw {
					args = append(args, "--raw")
				}
				out, err := execute(args...)
				if err != nil {
					t.Fatal(err)
				}
				var result transcriptSearchResult
				if err := json.Unmarshal([]byte(out), &result); err != nil {
					t.Fatal(err)
				}
				want := 0
				if raw {
					want = 1
				}
				if len(result.Matches) != want {
					t.Fatalf("ASSERT_NATIVE_PI_RESULTS_RAW_SEARCH: raw=%v matches=%d want=%d", raw, len(result.Matches), want)
				}
				if raw {
					m := result.Matches[0]
					if m.AgentID != "AAA" || m.Role != "toolResult" || m.SourcePath != side || m.EventID == "" {
						t.Fatalf("native result search provenance: %+v", m)
					}
				}
			}
		})
	}
}

func TestTranscriptNativeToolResultClassification(t *testing.T) {
	for name, record := range map[string]string{
		"valid":             `{"type":"toolResult","isSidechain":true,"agentId":"AAA","message":{"role":"toolResult","content":"payload"}}`,
		"missing-sidechain": `{"type":"toolResult","agentId":"AAA","message":{"role":"toolResult"}}`,
		"missing-owner":     `{"type":"toolResult","isSidechain":true,"message":{"role":"toolResult"}}`,
		"missing-role":      `{"type":"toolResult","isSidechain":true,"agentId":"AAA","message":{}}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "direct.output")
			writeTranscriptFile(t, path, record+"\n")
			want := schemaUnknown
			if name == "valid" {
				want = schemaPi
			}
			if got := classifyTranscript(path); got != want {
				t.Fatalf("ASSERT_NATIVE_PI_RESULTS_CLASSIFICATION: got=%s want=%s", got, want)
			}
		})
	}
}

func TestTranscriptNativeToolResultDoesNotMeasureUsage(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		parent, _ := nativeToolResultFixture(t, mixed)
		agents, err := buildTree(parent)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, a := range agents {
			if a.ID != "AAA" {
				continue
			}
			found = true
			wantCost, wantStatus := 0, "unavailable"
			if mixed {
				wantCost, wantStatus = 7, "complete"
			}
			if a.Cost != wantCost || a.CostStatus != wantStatus {
				t.Errorf("ASSERT_NATIVE_PI_RESULTS_NOT_USAGE: mixed=%v cost=%d status=%s want=%d/%s", mixed, a.Cost, a.CostStatus, wantCost, wantStatus)
			}
		}
		if !found {
			t.Fatal("missing spawned child")
		}
	}
}
