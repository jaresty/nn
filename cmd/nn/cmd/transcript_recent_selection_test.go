package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranscriptEventsLastSelectsFilteredTail(t *testing.T) {
	const assertion = "ASSERT_TRANSCRIPT_EVENTS_LAST_FILTERS_THEN_SELECTS_CANONICAL_TAIL"
	_, execute := setupNotebook(t)
	session := observabilitySession(t)

	out, err := execute("transcript", "events", session, "ROOT", "--errors-only", "--last", "1", "--select", "identity")
	if err != nil {
		t.Fatalf("%s: %v", assertion, err)
	}
	var page struct {
		Events []map[string]any `json:"events"`
		Query  struct {
			RequestedLast  int  `json:"requested_last"`
			MatchingEvents int  `json:"matching_events"`
			ReturnedEvents int  `json:"returned_events"`
			OlderMatching  bool `json:"older_matching_events"`
		} `json:"query"`
	}
	if err := json.Unmarshal([]byte(out), &page); err != nil {
		t.Fatalf("%s: decode: %v", assertion, err)
	}
	if len(page.Events) != 1 || page.Events[0]["ordinal"] != float64(5) {
		t.Fatalf("%s: filtered tail=%s", assertion, out)
	}
	if page.Query.RequestedLast != 1 || page.Query.MatchingEvents != 2 || page.Query.ReturnedEvents != 1 || !page.Query.OlderMatching {
		t.Fatalf("%s: receipt=%s", assertion, out)
	}
	if _, ok := page.Events[0]["message"]; ok {
		t.Fatalf("%s: selected facet leaked: %s", assertion, out)
	}

	out, err = execute("transcript", "events", session, "ROOT", "--last", "3", "--select", "identity")
	if err != nil {
		t.Fatalf("%s: unfiltered: %v", assertion, err)
	}
	page = struct {
		Events []map[string]any `json:"events"`
		Query  struct {
			RequestedLast  int  `json:"requested_last"`
			MatchingEvents int  `json:"matching_events"`
			ReturnedEvents int  `json:"returned_events"`
			OlderMatching  bool `json:"older_matching_events"`
		} `json:"query"`
	}{}
	if err := json.Unmarshal([]byte(out), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Events) != 3 || page.Events[0]["ordinal"] != float64(7) || page.Events[1]["ordinal"] != float64(8) || page.Events[2]["ordinal"] != float64(9) {
		t.Fatalf("%s: canonical tail=%s", assertion, out)
	}
	if page.Query.MatchingEvents != 9 || page.Query.ReturnedEvents != 3 || !page.Query.OlderMatching {
		t.Fatalf("%s: unfiltered receipt=%s", assertion, out)
	}

	for _, flags := range [][]string{
		{"--last", "0"},
		{"--last", "-1"},
		{"--last", "1", "--all"},
		{"--last", "1", "--event", "unknown"},
		{"--last", "1", "--summary", "timing"},
		{"--last", "1", "--at", "launch"},
	} {
		if _, err := execute(append([]string{"transcript", "events", session, "ROOT"}, flags...)...); err == nil {
			t.Fatalf("%s: accepted incompatible flags %v", assertion, flags)
		}
	}
	t.Log(assertion + ": PASS")
}

func descriptionFixture(t *testing.T) string {
	t.Helper()
	session := filepath.Join(t.TempDir(), "pi-description.jsonl")
	writeTranscriptFile(t, session,
		`{"type":"session","version":3,"id":"01a","cwd":"/x"}`+"\n"+
			`{"type":"message","id":"call-a","message":{"role":"assistant","content":[{"type":"toolCall","id":"agent-call-a","name":"Agent","arguments":{"description":"Same room","subagent_type":"general-purpose"}}]}}`+"\n"+
			`{"type":"message","id":"result-a","parentId":"call-a","message":{"role":"toolResult","toolCallId":"agent-call-a","toolName":"Agent","details":{"status":"background","agentId":"agent-a","description":"Same room","subagentType":"general-purpose"}}}`+"\n"+
			`{"type":"message","id":"call-b","message":{"role":"assistant","content":[{"type":"toolCall","id":"agent-call-b","name":"Agent","arguments":{"description":"Same room","subagent_type":"general-purpose"}}]}}`+"\n"+
			`{"type":"message","id":"result-b","parentId":"call-b","message":{"role":"toolResult","toolCallId":"agent-call-b","toolName":"Agent","details":{"status":"background","agentId":"agent-b","description":"Same room","subagentType":"general-purpose"}}}`+"\n")
	return session
}

func TestTranscriptTreeDescriptionSelectsAllExactMatches(t *testing.T) {
	const assertion = "ASSERT_TRANSCRIPT_TREE_DESCRIPTION_SELECTS_ALL_EXACT_METADATA_MATCHES"
	_, execute := setupNotebook(t)
	session := descriptionFixture(t)

	out, err := execute("transcript", "tree", session, "--json", "--description", "Same room", "--fields", "id,description,subtree_cost")
	if err != nil {
		t.Fatalf("%s: %v", assertion, err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("%s: decode: %v", assertion, err)
	}
	if len(rows) != 2 || rows[0]["id"] != "agent-a" || rows[1]["id"] != "agent-b" {
		t.Fatalf("%s: matches=%s", assertion, out)
	}
	for _, row := range rows {
		if len(row) != 3 || row["description"] != "Same room" {
			t.Fatalf("%s: field projection=%v", assertion, row)
		}
	}
	out, err = execute("transcript", "tree", session, "--json", "--description", "same room")
	if err != nil {
		t.Fatalf("%s: case-control: %v", assertion, err)
	}
	if strings.TrimSpace(out) != "[]" {
		t.Fatalf("%s: matching must be exact and case-sensitive: %s", assertion, out)
	}
	if _, err := execute("transcript", "tree", session, "--json", "--description", "Same room", "--agent", "agent-a"); err == nil {
		t.Fatalf("%s: accepted --description with --agent", assertion)
	}
	t.Log(assertion + ": PASS")
}
