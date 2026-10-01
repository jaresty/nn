package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func writePiForegroundFixture(t *testing.T, dir, name, parent, sessionName, assignment, stamp string) string {
	t.Helper()
	path := filepath.Join(dir, name+".jsonl")
	writeTranscriptFile(t, path,
		`{"type":"session","version":3,"id":"`+name+`","parentSession":"`+parent+`","timestamp":"`+stamp+`"}`+"\n"+
			`{"type":"custom","customType":"session_info","data":{"name":"`+sessionName+`"},"timestamp":"`+stamp+`"}`+"\n"+
			`{"type":"message","timestamp":"`+stamp+`","message":{"role":"user","content":[{"type":"text","text":"`+assignment+`"}]}}`+"\n"+
			`{"type":"message","timestamp":"`+stamp+`","message":{"role":"assistant","content":[{"type":"text","text":"done"}]}}`+"\n")
	return path
}

func piForegroundParentFixture(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "parent.jsonl")
	writeTranscriptFile(t, path,
		`{"type":"session","version":3,"id":"parent","timestamp":"2026-09-18T17:00:00Z"}`+"\n"+
			`{"type":"message","timestamp":"2026-09-18T17:01:00Z","message":{"role":"assistant","content":[{"type":"toolCall","id":"call-1","name":"Agent","arguments":{"description":"Adjudicate composed private B4","subagent_type":"integration-planner"}}]}}`+"\n"+
			`{"type":"message","timestamp":"2026-09-18T17:01:01Z","message":{"role":"toolResult","toolName":"Agent","toolCallId":"call-1","details":{"agentId":"1ccf743f-67ee-476","description":"Adjudicate composed private B4","subagentType":"integration-planner"},"content":[]}}`+"\n")
	return path
}

func TestPiForegroundOwnedSessionFallback(t *testing.T) {
	const assertion = "ASSERT_PI_FOREGROUND_OWNED_SESSION_FALLBACK"
	dir := t.TempDir()
	parent := piForegroundParentFixture(t, dir)
	resolved := writePiForegroundFixture(t, dir, "child", parent, "integration-planner#1ccf743f", "Adjudicate composed private B4", "2026-09-18T17:02:00Z")

	records, schema, detail, err := ledgerRecords(parent, "1ccf743f-67ee-476")
	if err != nil || schema != schemaPi || detail != "available" || len(records) != 2 {
		t.Fatalf("%s unique: records=%d schema=%q detail=%q err=%v", assertion, len(records), schema, detail, err)
	}
	resolution := transcriptChildDetailResolution(parent, "1ccf743f-67ee-476", schema, detail)
	if resolution.DetailSource != "owned_session_fallback" || resolution.Custody != "qualified_unique_join" || resolution.CandidateCount != 1 || resolution.ResolvedPath != canonicalPath(resolved) {
		t.Fatalf("%s resolution=%+v", assertion, resolution)
	}
	if !resolution.JoinEvidence["parent_session"] || !resolution.JoinEvidence["agent_type"] || !resolution.JoinEvidence["agent_id_prefix"] || !resolution.JoinEvidence["assignment"] || !resolution.JoinEvidence["launch_time"] {
		t.Fatalf("%s evidence=%v", assertion, resolution.JoinEvidence)
	}

	_, execute := setupNotebook(t)
	out, err := execute("transcript", "events", parent, "1ccf743f-67ee-476", "--all")
	if err != nil {
		t.Fatal(err)
	}
	var page ledgerPage
	if json.Unmarshal([]byte(out), &page) != nil || page.DetailSource != "owned_session_fallback" || page.Custody != "qualified_unique_join" || page.CandidateCount != 1 {
		t.Fatalf("%s page=%s", assertion, out)
	}
	text, err := execute("transcript", "events", parent, "1ccf743f-67ee-476", "--last", "2", "--format", "text")
	if err != nil || !strings.Contains(text, "detail source: owned_session_fallback · custody: qualified_unique_join · candidates: 1") {
		t.Fatalf("%s text=%q err=%v", assertion, text, err)
	}

	writePiForegroundFixture(t, dir, "duplicate", parent, "integration-planner#1ccf743f", "Adjudicate composed private B4", "2026-09-18T17:03:00Z")
	records, _, detail, err = ledgerRecords(parent, "1ccf743f-67ee-476")
	resolution = transcriptChildDetailResolution(parent, "1ccf743f-67ee-476", schemaPi, detail)
	if err != nil || detail != "unavailable" || len(records) != 0 || resolution.CandidateCount != 2 || resolution.DetailSource != "unavailable" {
		t.Fatalf("%s ambiguity records=%d detail=%q resolution=%+v err=%v", assertion, len(records), detail, resolution, err)
	}
}

func TestPiForegroundOwnedSessionFallbackRejectsMismatches(t *testing.T) {
	const assertion = "ASSERT_PI_FOREGROUND_FALLBACK_REJECTS_MISMATCHES"
	for _, tc := range []struct{ name, sessionName, assignment, stamp string }{
		{"type", "general-purpose#1ccf743f", "Adjudicate composed private B4", "2026-09-18T17:02:00Z"},
		{"prefix", "integration-planner#deadbeef", "Adjudicate composed private B4", "2026-09-18T17:02:00Z"},
		{"assignment", "integration-planner#1ccf743f", "different", "2026-09-18T17:02:00Z"},
		{"time", "integration-planner#1ccf743f", "Adjudicate composed private B4", "2026-09-18T16:00:00Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			parent := piForegroundParentFixture(t, dir)
			writePiForegroundFixture(t, dir, "child", parent, tc.sessionName, tc.assignment, tc.stamp)
			records, _, detail, err := ledgerRecords(parent, "1ccf743f-67ee-476")
			if err != nil || detail != "unavailable" || len(records) != 0 {
				t.Fatalf("%s %s: records=%d detail=%q err=%v", assertion, tc.name, len(records), detail, err)
			}
			if reason := transcriptDetailReason(parent, "1ccf743f-67ee-476", schemaPi, detail); !strings.Contains(reason, "locator_unavailable") {
				t.Fatalf("%s %s: reason=%q", assertion, tc.name, reason)
			}
		})
	}
}
