package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTranscriptPiEvidenceScope(t *testing.T) {
	for _, tc := range []struct {
		name                                                    string
		terminal, unavailable, resultOnly, foreignOnly, reverse bool
	}{
		{name: "terminal history", terminal: true},
		{name: "terminal unavailable", terminal: true, unavailable: true},
		{name: "background history"},
		{name: "background unavailable", unavailable: true},
		{name: "last record not latest time", terminal: true, reverse: true},
		{name: "tool results not usage", terminal: true, resultOnly: true},
		{name: "foreign events not usage", terminal: true, foreignOnly: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			side := filepath.Join(dir, "pi-subagents-test", "session", "tasks", "AAA.output")
			if !tc.unavailable {
				content := `{"type":"assistant","agentId":"AAA","message":{"role":"assistant","usage":{"input":10}}}` + "\n" + `{"type":"assistant","agentId":"AAA","message":{"role":"assistant","usage":{"input":20}}}` + "\n"
				if tc.resultOnly {
					content = `{"type":"toolResult","agentId":"AAA","message":{"role":"toolResult","usage":{"input":999}}}` + "\n"
				}
				if tc.foreignOnly {
					content = `{"type":"assistant","agentId":"BBB","message":{"role":"assistant","usage":{"input":999}}}` + "\n"
				}
				writeTranscriptFile(t, side, content)
			}
			base := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
			parent := filepath.Join(dir, "parent.jsonl")
			content := `{"type":"session"}` + "\n" + `{"type":"message","timestamp":"2026-09-06T00:00:00Z","message":{"role":"assistant","usage":{"input":2}}}` + "\n" + `{"type":"message","message":{"role":"toolResult","details":{"agentId":"AAA","status":"background","fullOutputPath":` + mustJSONString(t, side) + `}}}` + "\n"
			first, last := int64(1000), int64(10000)
			if tc.reverse {
				first, last = last, first
			}
			if tc.terminal {
				for _, offset := range []int64{first, last} {
					content += fmt.Sprintf(`{"type":"custom","customType":"subagents:record","data":{"id":"AAA","type":"general-purpose","status":"completed","result":"Blocked: worktree unavailable","startedAt":%d,"completedAt":%d}}`+"\n", base.UnixMilli()+offset, base.UnixMilli()+offset+500)
				}
			}
			writeTranscriptFile(t, parent, content)
			_, execute := setupNotebook(t)
			out, err := execute("transcript", "tree", parent, "--json", "--strict")
			if err != nil {
				t.Fatal(err)
			}
			var rows []map[string]json.RawMessage
			if err := json.Unmarshal([]byte(out), &rows); err != nil {
				t.Fatal(err)
			}
			if len(rows) != 2 {
				t.Fatalf("nodes: %s", out)
			}
			for _, row := range rows {
				var a agent
				encoded, _ := json.Marshal(row)
				json.Unmarshal(encoded, &a)
				raw, ok := row["evidence_scope"]
				if !ok {
					t.Fatal("ASSERT_PI_SCOPE_PRESENT: missing evidence_scope")
				}
				var scope struct {
					Status, Timestamps, Cost string
					SubtreeCost              string `json:"subtree_cost"`
					TerminalRecords          int    `json:"terminal_record_count"`
				}
				if err := json.Unmarshal(raw, &scope); err != nil {
					t.Fatal(err)
				}
				if scope.SubtreeCost != "subtree_aggregate" {
					t.Fatalf("ASSERT_PI_SCOPE_PROVENANCE: %+v", scope)
				}
				if a.ID == "ROOT" {
					if scope.Status != "unavailable" || scope.Timestamps != "root_message_history" || scope.Cost != "root_message_history" || scope.TerminalRecords != 0 {
						t.Fatalf("ASSERT_PI_SCOPE_PROVENANCE: root %+v", scope)
					}
					if a.Cost != 2 || a.Status != "" || a.Started != base.Format(time.RFC3339) {
						t.Fatalf("ASSERT_PI_SCOPE_PRESERVES_VALUES: root %+v", a)
					}
					continue
				}
				wantCost, wantStatus := "retained_sidechain_history", "complete"
				total := 30
				if tc.unavailable || tc.resultOnly || tc.foreignOnly {
					wantCost, wantStatus, total = "unavailable", "unavailable", 0
				}
				if scope.Cost != wantCost {
					t.Fatalf("ASSERT_PI_SCOPE_PROVENANCE: cost %+v", scope)
				}
				if a.Cost != total || a.SubtreeCost != total || a.CostStatus != wantStatus || a.ParentID != "ROOT" {
					t.Fatalf("ASSERT_PI_SCOPE_PRESERVES_VALUES: %+v", a)
				}
				if tc.terminal {
					if scope.Status != "last_terminal_record" || scope.Timestamps != "last_terminal_record" || scope.TerminalRecords != 2 {
						t.Fatalf("ASSERT_PI_SCOPE_PROVENANCE: terminal %+v", scope)
					}
					wantStart := time.UnixMilli(base.UnixMilli() + last).UTC().Format(time.RFC3339)
					if a.Status != "completed" || a.Started != wantStart || a.Ended != wantStart || a.Result != "Blocked: worktree unavailable" {
						t.Fatalf("ASSERT_PI_SCOPE_PRESERVES_VALUES: terminal %+v", a)
					}
				} else {
					if scope.Status != "background_spawn_record" || scope.Timestamps != "unavailable" || scope.TerminalRecords != 0 {
						t.Fatalf("ASSERT_PI_SCOPE_PROVENANCE: provisional %+v", scope)
					}
					if a.Status != "background" || a.Started != "" || a.Ended != "" {
						t.Fatalf("ASSERT_PI_SCOPE_PRESERVES_VALUES: provisional %+v", a)
					}
				}
			}
		})
	}
}

func TestTranscriptPiScopeDuplicateAndMissingTimes(t *testing.T) {
	path := writePiFixture(t, t.TempDir())
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	record := `{"type":"custom","customType":"subagents:record","data":{"id":"d1","status":"completed","result":"Blocked"}}` + "\n"
	if _, err := file.WriteString(record + record); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	_, execute := setupNotebook(t)
	out, err := execute("transcript", "tree", path, "--json", "--strict")
	if err != nil {
		t.Fatal(err)
	}
	var rows []agent
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatal(err)
	}
	for _, a := range rows {
		if a.ID != "d1" {
			continue
		}
		if a.EvidenceScope == nil || a.EvidenceScope.TerminalRecordCount != 3 || a.EvidenceScope.Timestamps != "last_terminal_record" {
			t.Fatalf("ASSERT_PI_SCOPE_RECORD_COUNT: %+v", a.EvidenceScope)
		}
		if a.Started != "" || a.Ended != "" || a.Status != "completed" || a.Result != "Blocked" {
			t.Fatalf("ASSERT_PI_SCOPE_PRESERVES_VALUES: %+v", a)
		}
		return
	}
	t.Fatal("missing child")
}

func TestTranscriptEvidenceScopeOmittedOtherSchemas(t *testing.T) {
	for _, fixture := range []func(*testing.T, string) string{writeSDKCLIFixture, writeClaudeCodeFixture} {
		path := fixture(t, t.TempDir())
		_, execute := setupNotebook(t)
		out, err := execute("transcript", "tree", path, "--json")
		if err != nil {
			t.Fatal(err)
		}
		var rows []map[string]json.RawMessage
		if err := json.Unmarshal([]byte(out), &rows); err != nil {
			t.Fatal(err)
		}
		for _, row := range rows {
			if _, ok := row["evidence_scope"]; ok {
				t.Fatal("ASSERT_PI_SCOPE_ONLY_PI: other schema changed")
			}
		}
	}
}
