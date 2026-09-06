package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// --- fixtures -------------------------------------------------------------

// sdk-cli fixture: a session file plus subagents/agent-*.jsonl + .meta.json,
// where the spawn edge is meta.toolUseId -> a tool_use block id in the parent.
func writeSDKCLIFixture(t *testing.T, dir string) string {
	t.Helper()
	session := filepath.Join(dir, "sess.jsonl")
	// parent (ROOT) issues a Task tool_use with id toolu_root, at t=100.
	writeTranscriptFile(t, session,
		`{"type":"assistant","uuid":"root-1","timestamp":"2026-09-06T00:00:00Z","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_root","name":"Task"}],"usage":{"input_tokens":10,"output_tokens":5}}}`+"\n")
	// child A: spawned by ROOT (meta.toolUseId = toolu_root).
	base := filepath.Join(dir, "sess", "subagents")
	writeTranscriptFile(t, filepath.Join(base, "agent-aaa.jsonl"),
		`{"type":"assistant","uuid":"a-1","timestamp":"2026-09-06T00:00:05Z","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_child","name":"Task"}],"usage":{"input_tokens":20,"output_tokens":10}}}`+"\n")
	writeTranscriptFile(t, filepath.Join(base, "agent-aaa.meta.json"),
		`{"agentType":"general-purpose","toolUseId":"toolu_root"}`)
	// grandchild B: spawned by child A (meta.toolUseId = toolu_child).
	writeTranscriptFile(t, filepath.Join(base, "agent-bbb.jsonl"),
		`{"type":"assistant","uuid":"b-1","timestamp":"2026-09-06T00:00:08Z","message":{"role":"assistant","content":[],"usage":{"input_tokens":30,"output_tokens":15}}}`+"\n")
	writeTranscriptFile(t, filepath.Join(base, "agent-bbb.meta.json"),
		`{"agentType":"general-purpose","toolUseId":"toolu_child"}`)
	return session
}

// pi fixture: single file, spawn edge = custom(subagents:record).parentId -> Agent toolCall record id.
func writePiFixture(t *testing.T, dir string) string {
	t.Helper()
	session := filepath.Join(dir, "pi.jsonl")
	writeTranscriptFile(t, session,
		`{"type":"session","version":3,"id":"01a","cwd":"/x"}`+"\n"+
			`{"type":"message","id":"m1","parentId":"a0","timestamp":"2026-09-06T00:00:00Z","message":{"role":"assistant","content":[{"type":"toolCall","id":"call_1","name":"Agent","arguments":{"subagent_type":"general-purpose"}}],"usage":{"input":10,"output":5}}}`+"\n"+
			`{"type":"custom","customType":"subagents:record","id":"c1","parentId":"m1","data":{"id":"d1","type":"general-purpose","status":"completed","result":"hello","startedAt":1788664537000,"completedAt":1788664539000}}`+"\n")
	return session
}

// claude-code fixture: inline Task; parent = assistant turn issuing the Task.
func writeClaudeCodeFixture(t *testing.T, dir string) string {
	t.Helper()
	session := filepath.Join(dir, "cc.jsonl")
	writeTranscriptFile(t, session,
		`{"type":"user","uuid":"u0","message":{"role":"user","content":"hi"}}`+"\n"+
			`{"type":"assistant","uuid":"u1","parentUuid":"u0","timestamp":"2026-09-06T00:00:00Z","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_cc","name":"Task","input":{"subagent_type":"general-purpose"}}],"usage":{"input_tokens":10,"output_tokens":5}}}`+"\n"+
			`{"type":"user","uuid":"u2","parentUuid":"u1","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_cc","content":"done"}]}}`+"\n")
	return session
}

// agentRow mirrors the normalized relation for JSON assertions.
type agentRow struct {
	ID         string `json:"id"`
	ParentID   string `json:"parent_id"`
	Type       string `json:"type"`
	Started    string `json:"started"`
	Ended      string `json:"ended"`
	Cost       int    `json:"cost"`
	SubtreeC   int    `json:"subtree_cost"`
	Status     string `json:"status"`
	Result     string `json:"result"`
}

func parseTreeJSON(t *testing.T, out string) []agentRow {
	t.Helper()
	var rows []agentRow
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("parse tree json: %v\noutput:\n%s", err, out)
	}
	return rows
}

func rowByID(rows []agentRow, id string) (agentRow, bool) {
	for _, r := range rows {
		if r.ID == id {
			return r, true
		}
	}
	return agentRow{}, false
}

// --- [7] sdk-cli spawn edge -----------------------------------------------

func TestTranscriptTreeSDKCLIParentEdges(t *testing.T) {
	dir := t.TempDir()
	session := writeSDKCLIFixture(t, dir)
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "tree", session, "--json")
	if err != nil {
		t.Fatalf("nn transcript tree: %v", err)
	}
	rows := parseTreeJSON(t, out)

	root, ok := rowByID(rows, "ROOT")
	if !ok {
		t.Fatalf("expected ROOT agent in rows: %+v", rows)
	}
	if root.ParentID != "" {
		t.Errorf("ROOT parent_id should be empty, got %q", root.ParentID)
	}
	a, ok := rowByID(rows, "aaa")
	if !ok {
		t.Fatalf("expected agent aaa in rows: %+v", rows)
	}
	if a.ParentID != "ROOT" {
		t.Errorf("agent aaa parent should be ROOT (via meta.toolUseId=toolu_root), got %q", a.ParentID)
	}
	b, ok := rowByID(rows, "bbb")
	if !ok {
		t.Fatalf("expected agent bbb in rows: %+v", rows)
	}
	if b.ParentID != "aaa" {
		t.Errorf("agent bbb parent should be aaa (recursive spawn via toolu_child), got %q", b.ParentID)
	}
}

// --- [8] pi spawn edge ----------------------------------------------------

func TestTranscriptTreePiParentEdge(t *testing.T) {
	dir := t.TempDir()
	session := writePiFixture(t, dir)
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "tree", session, "--json")
	if err != nil {
		t.Fatalf("nn transcript tree: %v", err)
	}
	rows := parseTreeJSON(t, out)

	sub, ok := rowByID(rows, "d1")
	if !ok {
		t.Fatalf("expected pi subagent d1 in rows: %+v", rows)
	}
	if sub.ParentID != "ROOT" {
		t.Errorf("pi subagent d1 parent should resolve to ROOT (custom.parentId=m1 -> Agent toolCall in root), got %q", sub.ParentID)
	}
	if sub.Result != "hello" {
		t.Errorf("pi subagent d1 result should be 'hello', got %q", sub.Result)
	}
	if sub.Status != "completed" {
		t.Errorf("pi subagent d1 status should be 'completed', got %q", sub.Status)
	}
}

// --- [9] claude-code inline Task (fixture-tested only) ---------------------

func TestTranscriptTreeClaudeCodeInlineTask(t *testing.T) {
	dir := t.TempDir()
	session := writeClaudeCodeFixture(t, dir)
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "tree", session, "--json")
	if err != nil {
		t.Fatalf("nn transcript tree: %v", err)
	}
	rows := parseTreeJSON(t, out)
	child, ok := rowByID(rows, "toolu_cc")
	if !ok {
		t.Fatalf("expected inline-Task child toolu_cc in rows: %+v", rows)
	}
	if child.ParentID != "ROOT" {
		t.Errorf("inline-Task child parent should be ROOT, got %q", child.ParentID)
	}
}

// --- [10] lifespan + cost + subtree_cost ----------------------------------

func TestTranscriptTreeCostAndSubtree(t *testing.T) {
	dir := t.TempDir()
	session := writeSDKCLIFixture(t, dir)
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "tree", session, "--json")
	if err != nil {
		t.Fatalf("nn transcript tree: %v", err)
	}
	rows := parseTreeJSON(t, out)

	b, _ := rowByID(rows, "bbb")
	if b.Cost != 45 { // 30+15
		t.Errorf("agent bbb cost should be 45, got %d", b.Cost)
	}
	a, _ := rowByID(rows, "aaa")
	// aaa cost 30, subtree = 30 + bbb(45) = 75
	if a.SubtreeC != 75 {
		t.Errorf("agent aaa subtree_cost should be 75 (30+45), got %d", a.SubtreeC)
	}
	root, _ := rowByID(rows, "ROOT")
	// ROOT cost 15, subtree = 15 + 30 + 45 = 90
	if root.SubtreeC != 90 {
		t.Errorf("ROOT subtree_cost should be 90 (15+30+45), got %d", root.SubtreeC)
	}
}

// --- [11] validation: cycle is rejected -----------------------------------

func TestTranscriptTreeRejectsCycle(t *testing.T) {
	dir := t.TempDir()
	// pi fixture where a custom record's parentId points to a non-existent id,
	// producing an unresolved (orphan) edge — validation must reject.
	session := filepath.Join(dir, "bad.jsonl")
	writeTranscriptFile(t, session,
		`{"type":"session","version":3,"id":"01a","cwd":"/x"}`+"\n"+
			`{"type":"custom","customType":"subagents:record","id":"c1","parentId":"does-not-exist","data":{"id":"d1","status":"completed"}}`+"\n")
	_, execute := setupNotebook(t)

	_, err := execute("transcript", "tree", session, "--json")
	if err == nil {
		t.Errorf("expected validation error for unresolved parent edge, got nil")
	}
}

// --- [12] show: lossless per-agent events ---------------------------------

func TestTranscriptShowAgentEvents(t *testing.T) {
	dir := t.TempDir()
	session := writePiFixture(t, dir)
	_, execute := setupNotebook(t)

	// show the pi subagent record d1 — must surface its schema-native result.
	out, err := execute("transcript", "show", session, "d1")
	if err != nil {
		t.Fatalf("nn transcript show: %v", err)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("expected schema-native result 'hello' in show output:\n%s", out)
	}
}

// --- text overview (descent-stack lifespan tree) --------------------------

func TestTranscriptTreeTextOverview(t *testing.T) {
	dir := t.TempDir()
	session := writeSDKCLIFixture(t, dir)
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "tree", session)
	if err != nil {
		t.Fatalf("nn transcript tree (text): %v", err)
	}
	// text overview shows the tree with indentation reflecting hierarchy:
	// ROOT, then aaa under it, then bbb under aaa.
	if !strings.Contains(out, "ROOT") || !strings.Contains(out, "aaa") || !strings.Contains(out, "bbb") {
		t.Errorf("expected ROOT/aaa/bbb in text overview:\n%s", out)
	}
	// bbb is deeper than aaa → more indentation.
	rootIdx := strings.Index(out, "ROOT")
	aaaIdx := strings.Index(out, "aaa")
	bbbIdx := strings.Index(out, "bbb")
	if !(rootIdx < aaaIdx && aaaIdx < bbbIdx) {
		t.Errorf("expected ROOT before aaa before bbb (depth order):\n%s", out)
	}
}
