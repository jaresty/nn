package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func transcriptDiagnosticsFixture(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	side := filepath.Join(dir, "pi-subagents-test", "session", "tasks", "AAA.output")
	payload := `{"ok":true,"data":{"omitted":true,"reason":"bounded","contentBlocks":1,"rawResultBytes":100,"contentSummary":[{"textOmitted":true}],"fullResultPath":"/unavailable/result.json","irrelevant":"first"}}`
	record := `{"type":"toolResult","isSidechain":true,"agentId":"AAA","message":{"role":"toolResult","toolName":"Bash","toolCallId":"diagnostics-call","isError":false,"content":[{"type":"text","text":` + mustJSONString(t, payload) + `}]}}`
	writeTranscriptFile(t, side, record+"\n")
	parent := filepath.Join(dir, "parent.jsonl")
	writeTranscriptFile(t, parent, `{"type":"session"}`+"\n"+
		`{"type":"message","message":{"role":"toolResult","details":{"agentId":"AAA","status":"background","fullOutputPath":`+mustJSONString(t, side)+`}}}`+"\n")
	return parent, side
}

func TestTranscriptEventsDiagnosticsWindowHidesPayload(t *testing.T) {
	session, _ := transcriptDiagnosticsFixture(t)
	_, execute := setupNotebook(t)
	out, err := execute("transcript", "events", session, "AAA", "--diagnostics", "--search", "omitted", "--kind", "tool_result", "--context", "0")
	if err != nil {
		t.Fatalf("ASSERT_DIAGNOSTICS_WINDOW_DETECTED: command failed: %v", err)
	}
	var page struct {
		Events []map[string]json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal([]byte(out), &page); err != nil {
		t.Fatalf("ASSERT_DIAGNOSTICS_WINDOW_DETECTED: invalid output: %v", err)
	}
	if len(page.Events) != 1 {
		t.Fatalf("ASSERT_DIAGNOSTICS_WINDOW_DETECTED: selected %d events, want 1", len(page.Events))
	}
	var diagnostic struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(page.Events[0]["diagnostics"], &diagnostic); err != nil || diagnostic.Status != transcriptArtifactDetected {
		t.Fatalf("ASSERT_DIAGNOSTICS_WINDOW_DETECTED: diagnostic=%s err=%v", page.Events[0]["diagnostics"], err)
	}
	if _, ok := page.Events[0]["payload"]; ok {
		t.Fatal("ASSERT_DIAGNOSTICS_WINDOW_NO_PAYLOAD: hidden payload was serialized")
	}
}

func TestTranscriptEventsDiagnosticsSnapshotBindsHiddenEvidence(t *testing.T) {
	session, side := transcriptDiagnosticsFixture(t)
	_, execute := setupNotebook(t)
	first, err := execute("transcript", "events", session, "AAA", "--diagnostics", "--last", "1")
	if err != nil {
		t.Fatalf("ASSERT_DIAGNOSTICS_SNAPSHOT_FIRST: %v", err)
	}
	var page struct {
		Snapshot string            `json:"snapshot"`
		Events   []json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal([]byte(first), &page); err != nil {
		t.Fatal(err)
	}
	var before struct {
		Diagnostics transcriptArtifactDiagnostic `json:"diagnostics"`
	}
	if len(page.Events) != 1 || json.Unmarshal(page.Events[0], &before) != nil || before.Diagnostics.Status != transcriptArtifactDetected {
		t.Fatalf("ASSERT_DIAGNOSTICS_SNAPSHOT_DIAGNOSIS: %s", first)
	}
	source, err := os.ReadFile(side)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(source), "first", "other", 1)
	if changed == string(source) {
		t.Fatal("ASSERT_DIAGNOSTICS_SNAPSHOT_FIXTURE")
	}
	if err := os.WriteFile(side, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := execute("transcript", "events", session, "AAA", "--diagnostics", "--last", "1")
	if err != nil {
		t.Fatal(err)
	}
	var after struct {
		Events []json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal([]byte(second), &after); err != nil {
		t.Fatal(err)
	}
	var diagnosis struct {
		Diagnostics transcriptArtifactDiagnostic `json:"diagnostics"`
	}
	if len(after.Events) != 1 || json.Unmarshal(after.Events[0], &diagnosis) != nil || diagnosis.Diagnostics.Status != before.Diagnostics.Status {
		t.Fatal("ASSERT_DIAGNOSTICS_SNAPSHOT_SAME_DIAGNOSIS")
	}
	if _, err := execute("transcript", "events", session, "AAA", "--diagnostics", "--last", "1", "--page", "2", "--snapshot", page.Snapshot); err == nil || !strings.Contains(err.Error(), "stale or mismatched") {
		t.Fatalf("ASSERT_DIAGNOSTICS_SNAPSHOT_HIDDEN_EVIDENCE: got %v", err)
	}
}

func TestTranscriptEventsDiagnosticsWindowSnapshotBindsHiddenEvidence(t *testing.T) {
	session, side := transcriptDiagnosticsFixture(t)
	_, execute := setupNotebook(t)
	args := []string{"transcript", "events", session, "AAA", "--diagnostics", "--search", "omitted", "--kind", "tool_result", "--context", "0"}
	first, err := execute(args...)
	if err != nil {
		t.Fatal(err)
	}
	var before struct {
		Snapshot string                       `json:"snapshot"`
		Events   []map[string]json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal([]byte(first), &before); err != nil || len(before.Events) != 1 {
		t.Fatalf("ASSERT_DIAGNOSTICS_WINDOW_SNAPSHOT_FIRST: %v", err)
	}
	source, err := os.ReadFile(side)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(side, []byte(strings.Replace(string(source), "first", "other", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := execute(args...)
	if err != nil {
		t.Fatal(err)
	}
	var after struct {
		Snapshot string                       `json:"snapshot"`
		Events   []map[string]json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal([]byte(second), &after); err != nil || len(after.Events) != 1 || string(before.Events[0]["diagnostics"]) != string(after.Events[0]["diagnostics"]) {
		t.Fatalf("ASSERT_DIAGNOSTICS_WINDOW_SNAPSHOT_SAME_DIAGNOSIS: %v", err)
	}
	if before.Snapshot == after.Snapshot {
		t.Fatal("ASSERT_DIAGNOSTICS_WINDOW_SNAPSHOT_HIDDEN_EVIDENCE: unchanged snapshot")
	}
	if _, err := execute(append(args, "--page", "2", "--snapshot", before.Snapshot)...); err == nil || !strings.Contains(err.Error(), "stale or mismatched") {
		t.Fatalf("ASSERT_DIAGNOSTICS_WINDOW_SNAPSHOT_STALE: %v", err)
	}
}

func TestTranscriptEventsDiagnosticsStatuses(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"not-detected", transcriptArtifactNotDetected},
		{"uninspected", transcriptArtifactUninspected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			session, side := transcriptDiagnosticsFixture(t)
			source, err := os.ReadFile(side)
			if err != nil {
				t.Fatal(err)
			}
			var changed string
			if tc.want == transcriptArtifactNotDetected {
				changed = strings.Replace(string(source), `\"omitted\":true`, `\"omitted\":false`, 1)
			} else {
				changed = strings.Replace(string(source), `"text":`+mustJSONString(t, `{"ok":true,"data":{"omitted":true,"reason":"bounded","contentBlocks":1,"rawResultBytes":100,"contentSummary":[{"textOmitted":true}],"fullResultPath":"/unavailable/result.json","irrelevant":"first"}}`), `"text":`+mustJSONString(t, "{broken"), 1)
			}
			if changed == string(source) {
				t.Fatal("ASSERT_DIAGNOSTICS_STATUS_FIXTURE")
			}
			if err := os.WriteFile(side, []byte(changed), 0600); err != nil {
				t.Fatal(err)
			}
			_, execute := setupNotebook(t)
			out, err := execute("transcript", "events", session, "AAA", "--diagnostics", "--last", "1")
			if err != nil {
				t.Fatal(err)
			}
			var page struct {
				Events []struct {
					Diagnostics transcriptArtifactDiagnostic `json:"diagnostics"`
				} `json:"events"`
			}
			if err := json.Unmarshal([]byte(out), &page); err != nil || len(page.Events) != 1 || page.Events[0].Diagnostics.Status != tc.want {
				t.Fatalf("ASSERT_DIAGNOSTICS_STATUS_%s: %v %s", tc.name, err, out)
			}
		})
	}
}

func TestTranscriptEventsDiagnosticsPreservesPayloadAndCompatibility(t *testing.T) {
	session, _ := transcriptDiagnosticsFixture(t)
	_, execute := setupNotebook(t)
	plain, err := execute("transcript", "events", session, "AAA", "--last", "1")
	if err != nil {
		t.Fatal(err)
	}
	var plainPage struct {
		Snapshot string                       `json:"snapshot"`
		Events   []map[string]json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal([]byte(plain), &plainPage); err != nil || len(plainPage.Events) != 1 {
		t.Fatalf("ASSERT_DIAGNOSTICS_UNFLAGGED_COMPATIBILITY: %v", err)
	}
	if _, ok := plainPage.Events[0]["diagnostics"]; ok {
		t.Fatal("ASSERT_DIAGNOSTICS_UNFLAGGED_COMPATIBILITY: diagnostic appeared without flag")
	}
	raw, err := execute("transcript", "events", session, "AAA", "--last", "1", "--payload")
	if err != nil {
		t.Fatal(err)
	}
	withDiagnostics, err := execute("transcript", "events", session, "AAA", "--last", "1", "--payload", "--diagnostics")
	if err != nil {
		t.Fatal(err)
	}
	var rawPage, diagnosticPage struct {
		Events []map[string]json.RawMessage `json:"events"`
	}
	if json.Unmarshal([]byte(raw), &rawPage) != nil || json.Unmarshal([]byte(withDiagnostics), &diagnosticPage) != nil || len(rawPage.Events) != 1 || len(diagnosticPage.Events) != 1 {
		t.Fatal("ASSERT_DIAGNOSTICS_PAYLOAD_PARITY: invalid event page")
	}
	if string(rawPage.Events[0]["payload"]) != string(diagnosticPage.Events[0]["payload"]) {
		t.Fatal("ASSERT_DIAGNOSTICS_PAYLOAD_PARITY: native payload changed")
	}
	var identity string
	if json.Unmarshal(diagnosticPage.Events[0]["event_id"], &identity) != nil || identity == "" {
		t.Fatal("ASSERT_DIAGNOSTICS_EXACT_EVENT: missing event_id")
	}
	exact, err := execute("transcript", "events", session, "AAA", "--event", identity, "--diagnostics")
	if err != nil || !strings.Contains(exact, `"diagnostics"`) {
		t.Fatalf("ASSERT_DIAGNOSTICS_EXACT_EVENT: %v %s", err, exact)
	}
	all, err := execute("transcript", "events", session, "AAA", "--all", "--diagnostics")
	if err != nil || !strings.Contains(all, `"diagnostics"`) {
		t.Fatalf("ASSERT_DIAGNOSTICS_ALL: %v %s", err, all)
	}
}

func TestTranscriptEventsDiagnosticsFragmentsCompleteEvent(t *testing.T) {
	session, side := transcriptDiagnosticsFixture(t)
	source, err := os.ReadFile(side)
	if err != nil {
		t.Fatal(err)
	}
	large := strings.Replace(string(source), "first", strings.Repeat("z", 60000), 1)
	if large == string(source) {
		t.Fatal("ASSERT_DIAGNOSTICS_FRAGMENT_FIXTURE")
	}
	if err := os.WriteFile(side, []byte(large), 0600); err != nil {
		t.Fatal(err)
	}
	_, execute := setupNotebook(t)
	events, page := ledgerAll(t, execute, session, "AAA", "--diagnostics", "--payload", "--last", "1")
	if page.Pages < 2 || len(events) != 1 {
		t.Fatalf("ASSERT_DIAGNOSTICS_FRAGMENT_COMPLETE: pages=%d events=%d", page.Pages, len(events))
	}
	var diagnostic transcriptArtifactDiagnostic
	encoded, _ := json.Marshal(events[0]["diagnostics"])
	if err := json.Unmarshal(encoded, &diagnostic); err != nil || diagnostic.Status != transcriptArtifactDetected {
		t.Fatalf("ASSERT_DIAGNOSTICS_FRAGMENT_DIAGNOSIS: %v %s", err, encoded)
	}
}

func TestTranscriptEventsDiagnosticsRejectsSpecialRoutes(t *testing.T) {
	session, _ := transcriptDiagnosticsFixture(t)
	_, execute := setupNotebook(t)
	for name, flags := range map[string][]string{
		"text":            {"--diagnostics", "--format", "text", "--last", "1"},
		"summary":         {"--diagnostics", "--summary", "usage"},
		"handoff":         {"--diagnostics", "--at", "launch"},
		"assignment":      {"--diagnostics", "--include-assignment"},
		"combined-errors": {"--diagnostics", "--format", "text", "--last", "1", "--include-errors", "1"},
	} {
		t.Run(name, func(t *testing.T) {
			args := append([]string{"transcript", "events", session, "AAA"}, flags...)
			if _, err := execute(args...); err == nil || !strings.Contains(err.Error(), "--diagnostics cannot be combined") {
				t.Fatalf("ASSERT_DIAGNOSTICS_SPECIAL_ROUTE_REJECT: %v", err)
			}
		})
	}
}

func TestTranscriptEventsDiagnosticsLastHidesPayload(t *testing.T) {
	session, _ := transcriptDiagnosticsFixture(t)
	_, execute := setupNotebook(t)
	out, err := execute("transcript", "events", session, "AAA", "--diagnostics", "--last", "1")
	if err != nil {
		t.Fatalf("ASSERT_DIAGNOSTICS_LAST_DETECTED: command failed: %v", err)
	}
	var page struct {
		Events []map[string]json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal([]byte(out), &page); err != nil {
		t.Fatalf("ASSERT_DIAGNOSTICS_LAST_DETECTED: invalid output: %v", err)
	}
	if len(page.Events) != 1 {
		t.Fatalf("ASSERT_DIAGNOSTICS_LAST_DETECTED: selected %d events, want 1", len(page.Events))
	}
	var diagnostic struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(page.Events[0]["diagnostics"], &diagnostic); err != nil || diagnostic.Status != transcriptArtifactDetected {
		t.Fatalf("ASSERT_DIAGNOSTICS_LAST_DETECTED: diagnostic=%s err=%v", page.Events[0]["diagnostics"], err)
	}
	if _, ok := page.Events[0]["payload"]; ok {
		t.Fatal("ASSERT_DIAGNOSTICS_LAST_NO_PAYLOAD: hidden payload was serialized")
	}
}
