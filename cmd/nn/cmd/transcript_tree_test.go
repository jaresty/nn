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
	ID            string `json:"id"`
	ParentID      string `json:"parent_id"`
	Type          string `json:"type"`
	Started       string `json:"started"`
	Ended         string `json:"ended"`
	Cost          int    `json:"cost"`
	SubtreeC      int    `json:"subtree_cost"`
	InputTokens   int    `json:"input_tokens"`
	OutputTokens  int    `json:"output_tokens"`
	CacheRead     int    `json:"cache_read_tokens"`
	CacheCreation int    `json:"cache_creation_tokens"`
	Status        string `json:"status"`
	Result        string `json:"result"`
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

func TestTranscriptTreePiTerminalEventChainPreservesSpawnParents(t *testing.T) {
	const assertion = "ASSERT_PI_TERMINAL_EVENT_CHAIN_IS_NOT_AGENT_PARENTAGE"
	dir := t.TempDir()
	session := filepath.Join(dir, "pi-terminal-chain.jsonl")
	writeTranscriptFile(t, session,
		`{"type":"session","version":3,"id":"01a","cwd":"/x"}`+"\n"+
			`{"type":"message","id":"m1","parentId":"a0","message":{"role":"assistant","content":[{"type":"toolCall","id":"call_1","name":"Agent","arguments":{}}]}}`+"\n"+
			`{"type":"message","id":"m2","parentId":"m1","message":{"role":"toolResult","toolCallId":"call_1","toolName":"Agent","content":[{"type":"text","text":"Agent started"}],"details":{"status":"background","agentId":"agent-a","subagentType":"general-purpose"}}}`+"\n"+
			`{"type":"message","id":"m3","parentId":"m2","message":{"role":"assistant","content":[{"type":"toolCall","id":"call_2","name":"Agent","arguments":{}}]}}`+"\n"+
			`{"type":"message","id":"m4","parentId":"m3","message":{"role":"toolResult","toolCallId":"call_2","toolName":"Agent","content":[{"type":"text","text":"Agent started"}],"details":{"status":"background","agentId":"agent-b","subagentType":"general-purpose"}}}`+"\n"+
			`{"type":"custom","customType":"subagents:record","id":"completion-a","parentId":"m4","data":{"id":"agent-a","type":"general-purpose","status":"completed","result":"A"}}`+"\n"+
			`{"type":"custom","customType":"subagents:record","id":"completion-b","parentId":"completion-a","data":{"id":"agent-b","type":"general-purpose","status":"completed","result":"B"}}`+"\n")
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "tree", session, "--json", "--strict")
	if err != nil {
		t.Fatalf("%s: strict tree rejected event-chain records: %v", assertion, err)
	}
	rows := parseTreeJSON(t, out)
	for _, id := range []string{"agent-a", "agent-b"} {
		got, ok := rowByID(rows, id)
		if !ok {
			t.Fatalf("%s: missing %s in %+v", assertion, id, rows)
		}
		if got.ParentID != "ROOT" {
			t.Fatalf("%s: %s parent=%q, want spawn owner ROOT", assertion, id, got.ParentID)
		}
		if got.Status != "completed" {
			t.Fatalf("%s: %s status=%q, want terminal completed", assertion, id, got.Status)
		}
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

// --- [21][22] typed token components (cache vs fresh vs output) -----------

func TestTranscriptTreeTypedTokens(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "sess.jsonl")
	// one assistant record with all four token classes present.
	writeTranscriptFile(t, session,
		`{"type":"assistant","uuid":"root-1","message":{"role":"assistant","content":[],"usage":{"input_tokens":2,"output_tokens":70,"cache_read_input_tokens":1000,"cache_creation_input_tokens":54220}}}`+"\n")
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "tree", session, "--json")
	if err != nil {
		t.Fatalf("nn transcript tree: %v", err)
	}
	rows := parseTreeJSON(t, out)
	root, ok := rowByID(rows, "ROOT")
	if !ok {
		t.Fatalf("expected ROOT: %+v", rows)
	}
	if root.InputTokens != 2 {
		t.Errorf("input_tokens want 2 got %d", root.InputTokens)
	}
	if root.OutputTokens != 70 {
		t.Errorf("output_tokens want 70 got %d", root.OutputTokens)
	}
	if root.CacheRead != 1000 {
		t.Errorf("cache_read_tokens want 1000 got %d", root.CacheRead)
	}
	if root.CacheCreation != 54220 {
		t.Errorf("cache_creation_tokens want 54220 got %d", root.CacheCreation)
	}
	// [22] cost must include cache tokens — not silently drop them.
	want := 2 + 70 + 1000 + 54220
	if root.Cost != want {
		t.Errorf("cost want %d (incl cache) got %d — cache tokens dropped", want, root.Cost)
	}
}

// --- [11] terminal record without spawn evidence --------------------------

func TestTranscriptTreePiTerminalWithoutSpawnUsesConservativeRoot(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "terminal-without-spawn.jsonl")
	writeTranscriptFile(t, session,
		`{"type":"session","version":3,"id":"01a","cwd":"/x"}`+"\n"+
			`{"type":"custom","customType":"subagents:record","id":"c1","parentId":"does-not-exist","data":{"id":"d1","status":"completed"}}`+"\n")
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "tree", session, "--json", "--strict")
	if err != nil {
		t.Fatalf("strict tree should accept conservative missing-spawn parentage: %v", err)
	}
	rows := parseTreeJSON(t, out)
	sub, ok := rowByID(rows, "d1")
	if !ok {
		t.Fatalf("expected terminal agent d1 in rows: %+v", rows)
	}
	if sub.ParentID != "ROOT" {
		t.Fatalf("terminal agent without spawn evidence parent=%q, want ROOT", sub.ParentID)
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

// --- [13] show filters noise: no attachment dumps, no tool-result payloads ---

func TestTranscriptShowFiltersNoise(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "sess.jsonl")
	writeTranscriptFile(t, session,
		`{"type":"assistant","uuid":"root-1","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_root","name":"Task"}]}}`+"\n")
	base := filepath.Join(dir, "sess", "subagents")
	// an agent file with: a spawn prompt, an attachment (noise), an assistant
	// tool_use, and a bulky tool_result (noise).
	writeTranscriptFile(t, filepath.Join(base, "agent-aaa.jsonl"),
		`{"type":"user","uuid":"a-0","message":{"role":"user","content":"Run the debrief protocol for the ls-clock session."}}`+"\n"+
			`{"type":"attachment","uuid":"a-1","attachment":{"type":"deferred_tools_delta","addedNames":["WebFetch","WebSearch","mcp__chrome__click"]}}`+"\n"+
			`{"type":"assistant","uuid":"a-2","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu_1","name":"Bash","input":{"command":"nn show --global"}}]}}`+"\n"+
			`{"type":"user","uuid":"a-3","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu_1","content":"BULKY_RESULT_PAYLOAD_THAT_IS_NOISE"}]}}`+"\n")
	writeTranscriptFile(t, filepath.Join(base, "agent-aaa.meta.json"),
		`{"agentType":"nn-hooks:nn-session-debrief","toolUseId":"toolu_root"}`)

	_, execute := setupNotebook(t)
	out, err := execute("transcript", "show", session, "aaa")
	if err != nil {
		t.Fatalf("nn transcript show: %v", err)
	}
	// meaningful content is present: the spawn prompt and the tool-call name.
	if !strings.Contains(out, "debrief protocol") {
		t.Errorf("expected spawn prompt in show output:\n%s", out)
	}
	if !strings.Contains(out, "Bash") {
		t.Errorf("expected tool-call name 'Bash' in show output:\n%s", out)
	}
	// noise is filtered: no attachment tool-name dump, no tool-result payload.
	if strings.Contains(out, "deferred_tools_delta") || strings.Contains(out, "mcp__chrome__click") {
		t.Errorf("attachment noise should be filtered from show output:\n%s", out)
	}
	if strings.Contains(out, "BULKY_RESULT_PAYLOAD_THAT_IS_NOISE") {
		t.Errorf("tool-result payload should be filtered from show output:\n%s", out)
	}
}

// piBackgroundSidechainFixture: a Pi parent whose Agent tool-result is a background
// spawn — status:background, agentId:A, and an "Output file: <path>" locator pointing
// at an external sidechain JSONL that holds A's real events. There is NO terminal
// subagents:record for A, so resolution must follow the locator.
func writePiBackgroundSidechainFixture(t *testing.T, dir string) string {
	t.Helper()
	// Real Pi active sidechains live OUTSIDE the session dir, under a temp
	// pi-subagents-*/<session-id>/tasks/<agent-id>.output layout.
	tmpRoot := t.TempDir()
	side := filepath.Join(tmpRoot, "pi-subagents-501", "sess-abc", "tasks", "79d3f783-b96d-4c7.output")
	// Real Pi sidechain records carry isSidechain:true + agentId.
	writeTranscriptFile(t, side,
		`{"isSidechain":true,"agentId":"79d3f783-b96d-4c7","type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"SIDECHAIN_HELLO"}],"usage":{"input":3,"output":4}}}`+"\n")

	session := filepath.Join(dir, "pi-bg.jsonl")
	// Real Pi background record: message.role=toolResult, toolName=Agent, structured
	// status/agentId under message.details, and the path ONLY in content[].text as an
	// "Output file: <path>" line (no structured output-file field; fullOutputPath null).
	bg := `{"type":"message","id":"m2","parentId":"m1","message":{` +
		`"role":"toolResult","toolCallId":"call_1","toolName":"Agent","isError":false,` +
		`"content":[{"type":"text","text":"Agent started in background.\nAgent ID: 79d3f783-b96d-4c7\nOutput file: ` + side + `\n\nYou will be notified when this agent completes.\nUse get_subagent_result to retrieve full results."}],` +
		`"details":{"status":"background","agentId":"79d3f783-b96d-4c7","description":"demo","subagentType":"general-purpose","fullOutputPath":null}}}`
	writeTranscriptFile(t, session,
		`{"type":"session","version":3,"id":"01b","cwd":"/x"}`+"\n"+
			`{"type":"message","id":"m1","parentId":"a0","message":{"role":"assistant","content":[{"type":"toolCall","id":"call_1","name":"Agent","arguments":{"subagent_type":"general-purpose"}}]}}`+"\n"+
			bg+"\n")
	return session
}

// Assertion [8]: show follows a Pi background Agent tool-result locator and renders
// the external sidechain events instead of "(no per-agent detail found ...)".
func TestTranscriptShowPiBackgroundSidechain(t *testing.T) {
	dir := t.TempDir()
	session := writePiBackgroundSidechainFixture(t, dir)
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "show", session, "79d3f783-b96d-4c7")
	if err != nil {
		t.Fatalf("nn transcript show: %v", err)
	}
	if strings.Contains(out, "no per-agent detail found") {
		t.Errorf("expected sidechain events, got not-found fallthrough:\n%s", out)
	}
	if !strings.Contains(out, "SIDECHAIN_HELLO") {
		t.Errorf("expected sidechain event text in show output:\n%s", out)
	}
}

// Assertion [9]: tree includes a Pi background child discovered via its Agent
// tool-result locator, even with no terminal subagents:record.
func TestTranscriptTreePiBackgroundSidechain(t *testing.T) {
	dir := t.TempDir()
	session := writePiBackgroundSidechainFixture(t, dir)
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "tree", session)
	if err != nil {
		t.Fatalf("nn transcript tree: %v", err)
	}
	if !strings.Contains(out, "79d3f783-b96d-4c7") {
		t.Errorf("expected background child in tree output:\n%s", out)
	}
}

// Assertion [10]: a locator that does not match Pi's tasks/<agent-id>.output layout
// (here a traversal to an arbitrary file) is rejected — child still appears provisional,
// no crash, no arbitrary file read.
func TestTranscriptShowPiBackgroundLocatorPathEscapeSafe(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "pi-escape.jsonl")
	writeTranscriptFile(t, session,
		`{"type":"session","version":3,"id":"01c","cwd":"/x"}`+"\n"+
			`{"type":"message","id":"m1","parentId":"a0","message":{"role":"assistant","content":[{"type":"toolCall","id":"call_1","name":"Agent","arguments":{}}]}}`+"\n"+
			`{"type":"message","id":"m2","parentId":"m1","message":{"role":"toolResult","toolCallId":"call_1","toolName":"Agent","content":[{"type":"text","text":"Output file: ../../../../etc/passwd"}],"details":{"status":"background","agentId":"AAA"}}}`+"\n")
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "show", session, "AAA")
	if err != nil {
		t.Fatalf("nn transcript show: %v", err)
	}
	if strings.Contains(out, "root:") || strings.Contains(out, "/bin/") {
		t.Errorf("unauthenticated locator must not read an arbitrary file:\n%s", out)
	}
	if !strings.Contains(out, "background") {
		t.Errorf("expected provisional background metadata for unauthenticated locator:\n%s", out)
	}
}

// Assertion [11]: a path whose filename does not match the requested agent id is rejected
// even when it sits in a valid pi-subagents tasks dir (identity binding).
func TestTranscriptShowPiBackgroundLocatorAgentMismatchRejected(t *testing.T) {
	dir := t.TempDir()
	tmpRoot := t.TempDir()
	// A real-looking sidechain, but named for a DIFFERENT agent id.
	wrong := filepath.Join(tmpRoot, "pi-subagents-501", "sess-x", "tasks", "OTHER.output")
	writeTranscriptFile(t, wrong,
		`{"isSidechain":true,"agentId":"OTHER","type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"WRONG_AGENT_SECRET"}]}}`+"\n")
	session := filepath.Join(dir, "pi-mismatch.jsonl")
	writeTranscriptFile(t, session,
		`{"type":"session","version":3,"id":"01d","cwd":"/x"}`+"\n"+
			`{"type":"message","id":"m1","parentId":"a0","message":{"role":"assistant","content":[{"type":"toolCall","id":"call_1","name":"Agent","arguments":{}}]}}`+"\n"+
			`{"type":"message","id":"m2","parentId":"m1","message":{"role":"toolResult","toolCallId":"call_1","toolName":"Agent","content":[{"type":"text","text":"Output file: `+wrong+`"}],"details":{"status":"background","agentId":"AAA"}}}`+"\n")
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "show", session, "AAA")
	if err != nil {
		t.Fatalf("nn transcript show: %v", err)
	}
	if strings.Contains(out, "WRONG_AGENT_SECRET") {
		t.Errorf("a path named for a different agent id must be rejected:\n%s", out)
	}
	if !strings.Contains(out, "background") {
		t.Errorf("expected provisional background metadata for mismatched locator:\n%s", out)
	}
}
