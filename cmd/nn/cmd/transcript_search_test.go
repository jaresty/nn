package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranscriptSearchMeaningfulJSONProvenanceAndBounds(t *testing.T) {
	const assertion = "ASSERT_TRANSCRIPT_SEARCH_RETURNS_BOUNDED_MEANINGFUL_PROVENANCE"
	dir := t.TempDir()
	session := filepath.Join(dir, "pi.jsonl")
	writeTranscriptFile(t, session,
		`{"type":"session","version":3,"id":"sess-pi"}`+"\n"+
			`{"type":"message","id":"e1","agentId":"AAA","timestamp":"2026-09-06T10:00:00Z","message":{"role":"assistant","content":[{"type":"text","text":"Needle first"}]}}`+"\n"+
			`{"type":"message","id":"e2","agentId":"AAA","timestamp":"2026-09-06T11:00:00Z","message":{"role":"assistant","content":[{"type":"text","text":"needle second"}]}}`+"\n"+
			`{"type":"message","id":"noise","agentId":"AAA","timestamp":"2026-09-06T12:00:00Z","message":{"role":"toolResult","content":[{"type":"tool_result","text":"NOISE_ONLY_NEEDLE"}]}}`+"\n")
	_, execute := setupNotebook(t)
	out, err := execute("transcript", "search", "NEEDLE", "--session", session, "--agent", "AAA", "--limit", "1", "--json")
	if err != nil {
		t.Fatalf("%s: %v", assertion, err)
	}
	var got struct {
		Matches []struct {
			Session    string `json:"session"`
			AgentID    string `json:"agent_id"`
			EventID    string `json:"event_id"`
			Timestamp  string `json:"timestamp"`
			Role       string `json:"role"`
			Excerpt    string `json:"excerpt"`
			SourcePath string `json:"source_path"`
		} `json:"matches"`
		Returned  int  `json:"returned"`
		Truncated bool `json:"truncated"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("%s: invalid JSON: %v\n%s", assertion, err, out)
	}
	if len(got.Matches) != 1 || got.Returned != 1 || !got.Truncated {
		t.Fatalf("%s: matches/returned/truncated = %d/%d/%v", assertion, len(got.Matches), got.Returned, got.Truncated)
	}
	m := got.Matches[0]
	if m.Session == "" || m.AgentID != "AAA" || m.EventID != "e1" || m.Timestamp == "" || m.Role != "assistant" || m.Excerpt != "Needle first" || m.SourcePath != session {
		t.Fatalf("%s: incomplete/wrong provenance: %+v", assertion, m)
	}
}

func TestTranscriptSearchMeaningfulExcludesRawNoise(t *testing.T) {
	const assertion = "ASSERT_TRANSCRIPT_SEARCH_RAW_MODE_IS_EXPLICIT"
	dir := t.TempDir()
	session := filepath.Join(dir, "pi.jsonl")
	writeTranscriptFile(t, session,
		`{"type":"session","version":3,"id":"sess-pi"}`+"\n"+
			`{"type":"message","timestamp":"2026-09-06T10:00:00Z","message":{"role":"toolResult","content":[{"type":"text","text":"RAW_ONLY_NEEDLE"}]}}`+"\n")
	_, execute := setupNotebook(t)
	out, err := execute("transcript", "search", "RAW_ONLY_NEEDLE", "--session", session, "--json")
	if err != nil {
		t.Fatalf("%s: %v", assertion, err)
	}
	if strings.Contains(out, "RAW_ONLY_NEEDLE") {
		t.Fatalf("%s: meaningful mode leaked tool-result payload: %s", assertion, out)
	}
	out, err = execute("transcript", "search", "RAW_ONLY_NEEDLE", "--session", session, "--raw", "--json")
	if err != nil || !strings.Contains(out, "RAW_ONLY_NEEDLE") {
		t.Fatalf("%s: raw mode did not find payload: err=%v out=%s", assertion, err, out)
	}
	if !strings.Contains(out, `"event_id": "record:2"`) {
		t.Fatalf("%s: ID-less event lacks deterministic ordinal provenance: %s", assertion, out)
	}
}
