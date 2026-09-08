package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// twoSessionDir builds a dir with two sessions of different mtimes:
// "old.jsonl" (older) and an sdk-cli session "new.jsonl" (newer) with subagents.
func twoSessionDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	old := filepath.Join(dir, "old.jsonl")
	writeTranscriptFile(t, old, `{"type":"assistant","uuid":"o1","message":{"role":"assistant","content":[],"usage":{"input_tokens":5,"output_tokens":5}}}`+"\n")

	// sdk-cli session with a small tree.
	writeSDKCLIFixture(t, dir) // creates sess.jsonl + subagents

	// set mtimes: old older than sess.
	oldTime := time.Now().Add(-2 * time.Hour)
	newTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(old, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(dir, "sess.jsonl"), newTime, newTime); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestTranscriptLsEmptyDiscovery(t *testing.T) {
	const notice = "No matching transcript sessions found.\n"
	childOnly := t.TempDir()
	writeTranscriptFile(t, filepath.Join(childOnly, "subagents", "agent-aaa.jsonl"),
		`{"type":"assistant","message":{"role":"assistant","content":"child"}}`+"\n")
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"empty directory", []string{t.TempDir()}},
		{"children excluded", []string{childOnly}},
		{"all filtered", []string{twoSessionDir(t), "--before", "1970-01-01T00:00:00Z"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, execute := setupNotebook(t)
			args := append([]string{"transcript", "ls"}, tc.args...)
			out, err := execute(args...)
			if err != nil || out != notice {
				t.Errorf("ASSERT_TRANSCRIPT_LS_EMPTY_TEXT_IS_EXPLICIT: output=%q error=%v", out, err)
			}
			out, err = execute(append(args, "--json")...)
			if err != nil || out != "[]\n" {
				t.Errorf("ASSERT_TRANSCRIPT_LS_EMPTY_JSON_STAYS_ARRAY: output=%q error=%v", out, err)
			}
		})
	}
}

func TestTranscriptLsErrorsAreNotEmptyDiscovery(t *testing.T) {
	for _, args := range [][]string{
		{filepath.Join(t.TempDir(), "missing")},
		{t.TempDir(), "--before", "invalid"},
	} {
		for _, asJSON := range []bool{false, true} {
			_, execute := setupNotebook(t)
			command := append([]string{"transcript", "ls"}, args...)
			if asJSON {
				command = append(command, "--json")
			}
			out, err := execute(command...)
			if err == nil || strings.Contains(out, "No matching transcript sessions found.") || strings.TrimSpace(out) == "[]" {
				t.Fatalf("discovery error became empty success: args=%v output=%q error=%v", command, out, err)
			}
		}
	}
}

// Assertion [14]: ls lists sessions most-recent-first with schema, agent count, cost.
func TestTranscriptLsListsRecentFirst(t *testing.T) {
	dir := twoSessionDir(t)
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "ls", dir)
	if err != nil {
		t.Fatalf("nn transcript ls: %v", err)
	}
	// newer sdk-cli session ("sess") should appear before the older one ("old").
	sessIdx := strings.Index(out, "sess")
	oldIdx := strings.Index(out, "old")
	if sessIdx == -1 || oldIdx == -1 {
		t.Fatalf("expected both sessions listed:\n%s", out)
	}
	if sessIdx > oldIdx {
		t.Errorf("expected newer session (sess) listed before older (old):\n%s", out)
	}
	if !strings.Contains(out, "sdk-cli") {
		t.Errorf("expected schema label in ls output:\n%s", out)
	}
}

// Assertion [15]: each row includes a compact inline mini-tree preview.
func TestTranscriptLsMiniTreePreview(t *testing.T) {
	dir := twoSessionDir(t)
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "ls", dir)
	if err != nil {
		t.Fatalf("nn transcript ls: %v", err)
	}
	// the sdk-cli fixture has ROOT + aaa + bbb; a mini-tree preview names ROOT.
	if !strings.Contains(out, "ROOT") {
		t.Errorf("expected a mini-tree preview containing ROOT:\n%s", out)
	}
}

// Assertion [16]: --limit caps rows; --before pages older.
func TestTranscriptLsLimitAndBefore(t *testing.T) {
	dir := twoSessionDir(t)
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "ls", dir, "--limit", "1")
	if err != nil {
		t.Fatalf("nn transcript ls --limit: %v", err)
	}
	// only the newest (sess) should appear, not old.
	if strings.Contains(out, "old.jsonl") || strings.Contains(out, "old ") {
		t.Errorf("--limit 1 should exclude the older session:\n%s", out)
	}

	// --before 90m ago should exclude the newer (1h-old) session, include old (2h).
	before := time.Now().Add(-90 * time.Minute).UTC().Format(time.RFC3339)
	out2, err := execute("transcript", "ls", dir, "--before", before)
	if err != nil {
		t.Fatalf("nn transcript ls --before: %v", err)
	}
	if !strings.Contains(out2, "old") {
		t.Errorf("--before should include the older session:\n%s", out2)
	}
	if strings.Contains(out2, "sess.jsonl") {
		t.Errorf("--before 90m should exclude the 1h-old session:\n%s", out2)
	}
}

// Assertion [17]: --json emits structured session rows.
func TestTranscriptLsConversationMetadata(t *testing.T) {
	const (
		labelAssertion      = "ASSERT_TRANSCRIPT_LS_USES_LATEST_SUBSTANTIAL_LABEL"
		openingAssertion    = "ASSERT_TRANSCRIPT_LS_PRESERVES_OPENING_LABEL"
		provenanceAssertion = "ASSERT_TRANSCRIPT_LS_QUALIFIES_RECENT_LABEL_PROVENANCE"
		kindAssertion       = "ASSERT_TRANSCRIPT_LS_CLASSIFIES_RETAINED_CONVERSATIONS_WITHOUT_GUESSED_OWNER"
		identityAssertion   = "ASSERT_TRANSCRIPT_LS_PRESERVES_SESSION_AND_CANONICAL_PATH"
		windowAssertion     = "ASSERT_TRANSCRIPT_LS_OPEN_WINDOW_STATUS_IS_UNAVAILABLE"
		compatAssertion     = "ASSERT_TRANSCRIPT_LS_ADDITIVE_FIELDS_PRESERVE_CURSOR_AND_SUMMARY"
	)
	dir := t.TempDir()
	path := filepath.Join(dir, "conversation.jsonl")
	writeTranscriptFile(t, path,
		`{"type":"session","version":3,"id":"conversation","cwd":"/workspace"}`+"\n"+
			`{"type":"message","id":"opening","message":{"role":"user","content":[{"type":"text","text":"Design readable transcript discovery"}]}}`+"\n"+
			`{"type":"message","id":"recent","message":{"role":"user","content":[{"type":"text","text":"Show the latest meaningful conversation work"}]}}`+"\n"+
			`{"type":"message","id":"ack","message":{"role":"user","content":[{"type":"text","text":"ok, let's try it"}]}}`+"\n")

	_, execute := setupNotebook(t)
	out, err := execute("transcript", "ls", dir, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Cursor           string          `json:"cursor"`
		Session          string          `json:"session"`
		Path             string          `json:"path"`
		Label            string          `json:"label"`
		OpeningLabel     string          `json:"opening_label"`
		LabelProvenance  string          `json:"label_provenance"`
		ConversationKind string          `json:"conversation_kind"`
		OwnerSession     *string         `json:"owner_session"`
		OpenWindowStatus string          `json:"open_window_status"`
		Summary          *sessionSummary `json:"summary"`
	}
	if err := json.Unmarshal([]byte(out), &rows); err != nil || len(rows) != 1 {
		t.Fatalf("decode rows: %v output=%s", err, out)
	}
	row := rows[0]
	if row.Label != "Show the latest meaningful conversation work" {
		t.Errorf("%s: label=%q", labelAssertion, row.Label)
	}
	if row.OpeningLabel != "Design readable transcript discovery" {
		t.Errorf("%s: opening=%q", openingAssertion, row.OpeningLabel)
	}
	if row.LabelProvenance != "recent" {
		t.Errorf("%s: provenance=%q", provenanceAssertion, row.LabelProvenance)
	}
	if row.ConversationKind != "conversation" || row.OwnerSession != nil {
		t.Errorf("%s: kind=%q owner=%v", kindAssertion, row.ConversationKind, row.OwnerSession)
	}
	absolute, _ := filepath.Abs(path)
	if row.Session != "conversation" || row.Path != absolute {
		t.Errorf("%s: session=%q path=%q want=%q", identityAssertion, row.Session, row.Path, absolute)
	}
	if row.OpenWindowStatus != "unavailable" {
		t.Errorf("%s: status=%q", windowAssertion, row.OpenWindowStatus)
	}
	if row.Cursor == "" || row.Summary == nil {
		t.Errorf("%s: cursor=%q summary=%v", compatAssertion, row.Cursor, row.Summary)
	}
}

func TestTranscriptLsClassifiesPiAgentExecutionAsSidechain(t *testing.T) {
	const assertion = "ASSERT_TRANSCRIPT_LS_PI_AGENT_EXECUTION_IS_QUALIFIED_SIDECHAIN"
	dir := filepath.Join(t.TempDir(), "pi-agent-child-id")
	path := filepath.Join(dir, "child.jsonl")
	writeTranscriptFile(t, path,
		`{"type":"session","version":3,"id":"child","cwd":"`+dir+`"}`+"\n"+
			`{"type":"message","id":"opening","message":{"role":"user","content":[{"type":"text","text":"Child assignment"}]}}`+"\n")
	rows, err := listSessions(filepath.Dir(dir), 0, time.Time{})
	if err != nil || len(rows) != 1 {
		t.Fatalf("%s: rows=%v error=%v", assertion, rows, err)
	}
	if rows[0].ConversationKind != "sidechain" || rows[0].OwnerSession != nil {
		t.Fatalf("%s: kind=%q owner=%v", assertion, rows[0].ConversationKind, rows[0].OwnerSession)
	}
}

func TestTranscriptLsJSON(t *testing.T) {
	dir := twoSessionDir(t)
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "ls", dir, "--json")
	if err != nil {
		t.Fatalf("nn transcript ls --json: %v", err)
	}
	var rows []struct {
		Session     string `json:"session"`
		Path        string `json:"path"`
		Modified    string `json:"modified"`
		Schema      string `json:"schema"`
		AgentCount  int    `json:"agent_count"`
		TotalCost   int    `json:"total_cost"`
		TreePreview string `json:"tree_preview"`
	}
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("parse ls json: %v\n%s", err, out)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 session rows, got %d:\n%s", len(rows), out)
	}
	// newest first
	if rows[0].Schema != "sdk-cli" {
		t.Errorf("expected newest row to be sdk-cli, got %q", rows[0].Schema)
	}
	if rows[0].AgentCount != 3 { // ROOT + aaa + bbb
		t.Errorf("expected sdk-cli agent_count 3, got %d", rows[0].AgentCount)
	}
	if rows[0].TreePreview == "" {
		t.Errorf("expected non-empty tree_preview in json row")
	}
}
