package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTranscriptTreeHallwaySummaryAndPaging(t *testing.T) {
	const summaryA = "ASSERT_TRANSCRIPT_TREE_NATIVE_HALLWAY_SUMMARY"
	const pageA = "ASSERT_TRANSCRIPT_TREE_PARENT_FILTER_PRECEDES_PAGINATION"
	const cursorA = "ASSERT_TRANSCRIPT_TREE_PARENT_CURSOR_FAILS_CLOSED"
	dir := t.TempDir()
	session := writeSDKCLIFixture(t, dir)
	writeTranscriptFile(t, dir+"/sess/subagents/agent-ccc.jsonl", `{"type":"assistant","uuid":"c"}`+"\n")
	writeTranscriptFile(t, dir+"/sess/subagents/agent-ccc.meta.json", `{"agentType":"general-purpose","toolUseId":"toolu_root"}`)
	_, execute := setupNotebook(t)
	out, err := execute("transcript", "tree", session, "--summary", "--json")
	var s treeHallwaySummary
	if err != nil || json.Unmarshal([]byte(out), &s) != nil || s.TotalAgents != 4 || s.DirectChildren != 2 || s.NestedEdges != 1 || s.OmittedAgents != 0 {
		t.Fatalf("%s: %#v out=%s err=%v", summaryA, s, out, err)
	}
	out, err = execute("transcript", "tree", session, "--parent", "ROOT", "--limit", "1", "--json")
	var p treeChildPage
	if err != nil || json.Unmarshal([]byte(out), &p) != nil || p.TotalChildren != 2 || p.Returned != 1 || len(p.Children) != 1 || p.Children[0].ID != "aaa" {
		t.Fatalf("%s: %#v out=%s err=%v", pageA, p, out, err)
	}
	if _, err = execute("transcript", "tree", session, "--parent", "aaa", "--cursor", p.NextCursor, "--json"); err == nil || !strings.Contains(err.Error(), "stale or mismatched cursor") {
		t.Fatalf("%s: %v", cursorA, err)
	}
	if _, err = execute("transcript", "tree", session, "--summary"); err == nil {
		t.Fatal("ASSERT_TRANSCRIPT_TREE_HALLWAY_MODES_REQUIRE_JSON")
	}
}
