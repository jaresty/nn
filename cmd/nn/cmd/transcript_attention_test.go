package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func attentionRecord(t *testing.T, ordinal int, role, content string) ledgerRecord {
	t.Helper()
	var r rawRecord
	s := fmt.Sprintf(`{"type":"message","agentId":"A","message":{"role":%q,"content":%s}}`, role, content)
	if e := json.Unmarshal([]byte(s), &r); e != nil {
		t.Fatal(e)
	}
	r.RecordOrdinal = ordinal
	return ledgerRecord{Record: r, Path: "/retained/source.jsonl"}
}
func attentionCall(id, name, args string) string {
	return fmt.Sprintf(`[{"type":"toolCall","id":%q,"name":%q,"arguments":%s}]`, id, name, args)
}

func TestAttentionPublishedSurfaces(t *testing.T) {
	t.Log("procedure: TestAttentionPublishedSurfaces; assertion: P7_SURFACES")
	_, execute := setupNotebook(t)
	for ref, phrases := range map[string][]string{
		"attention": {"Awaiting return badge", "Room evidence", "More → Attention signals:** offer bounded", "including returned rooms", "No signal does not mean healthy", "snapshot", "effective"},
		"navigate":  {"More → Attention signals", "Load **attention**"},
		"review":    {"Load **attention**", "membership/order"},
		"rooms":     {"Load **attention**", "Inspect evidence"},
	} {
		out, e := execute("skills", "get", "nn-transcript", "--reference", ref)
		if e != nil {
			t.Fatalf("P7_SURFACES FAIL: %v", e)
		}
		for _, phrase := range phrases {
			if !strings.Contains(out, phrase) {
				t.Fatalf("P7_SURFACES FAIL: %s missing %s", ref, phrase)
			}
		}
	}
	t.Log("P7_SURFACES PASS")
}

func TestAttentionMetrics(t *testing.T) {
	t.Log("procedure: TestAttentionMetrics; assertion: P2_METRICS")
	command := attentionCall("c", "bash", `{"command":"echo x"}`)
	records := []ledgerRecord{
		attentionRecord(t, 1, "user", `"instruction"`),
		attentionRecord(t, 2, "assistant", command),
		attentionRecord(t, 3, "assistant", command),
		attentionRecord(t, 4, "user", `[{"type":"tool_result","tool_use_id":"c","content":"x"}]`),
		attentionRecord(t, 5, "assistant", attentionCall("e", "edit", `{"path":"x.go","oldText":"x","newText":"y"}`)),
		attentionRecord(t, 6, "assistant", attentionCall("r", "read", `{"path":"x.go"}`)),
	}
	m, w, ev, e := collectAttention(records, "available", "A", 100)
	if e != nil || m.Commands != 1 || m.Edits != 1 || m.Neutral != 1 || m.Duplicates != 1 || m.Unknown != 0 || w.Selected != 5 || len(ev) != 5 {
		t.Fatalf("P2_METRICS FAIL: %+v %+v %v", m, w, e)
	}
	m, w, _, e = collectAttention(records, "available", "A", 2)
	if e != nil || m.Commands != 0 || m.Edits != 1 || w.Selected != 2 || w.Earlier != 4 {
		t.Fatalf("P2_METRICS FAIL: boundary %+v %+v %v", m, w, e)
	}
	records = append(records, attentionRecord(t, 7, "assistant", attentionCall("u", "unknownTool", `{}`)))
	m, _, _, _ = collectAttention(records, "available", "A", 100)
	if m.Unknown != 1 {
		t.Fatal("P2_METRICS FAIL: unknown tool silently classified")
	}
	records = append(records, attentionRecord(t, 8, "assistant", attentionCall("c", "bash", `{"command":"changed"}`)))
	m, _, _, _ = collectAttention(records, "available", "A", 100)
	if m.Unknown != 2 {
		t.Fatal("P2_METRICS FAIL: conflicting invocation ID accepted")
	}
	t.Log("P2_METRICS PASS")
}

func attentionFixture(t *testing.T, schema string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	id := "ROOT"
	prefix := ""
	typ := "assistant"
	block := "tool_use"
	arg := "input"
	if schema == "pi" {
		prefix = `{"type":"session"}` + "\n"
		typ = "message"
		block = "toolCall"
		arg = "arguments"
	}
	if schema == "sdk-cli" {
		path = writeSDKCLIFixture(t, dir)
		id = "aaa"
		pathChild := filepath.Join(dir, "sess", "subagents", "agent-aaa.jsonl")
		var b strings.Builder
		for i := 0; i < 40; i++ {
			fmt.Fprintf(&b, `{"type":"assistant","uuid":"u%d","message":{"role":"assistant","content":[{"type":"tool_use","id":"cmd%d","name":"Bash","input":{"command":"echo x"}}]}}`+"\n", i, i)
		}
		// A foreign explicitly owned record must not enter this agent's counts.
		b.WriteString(`{"type":"assistant","uuid":"foreign","agentId":"BBB","message":{"role":"assistant","content":[{"type":"tool_use","id":"foreign","name":"Edit","input":{"file_path":"x"}}]}}` + "\n")
		writeTranscriptFile(t, pathChild, b.String())
		return path, id
	}
	var b strings.Builder
	b.WriteString(prefix)
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&b, `{"type":%q,"uuid":"u%d","message":{"role":"assistant","content":[{"type":%q,"id":"cmd%d","name":"Bash",%q:{"command":"echo x"}}]}}`+"\n", typ, i, block, i, arg)
	}
	writeTranscriptFile(t, path, b.String())
	return path, id
}

func TestAttentionClaudeAndPi(t *testing.T) {
	t.Log("procedure: TestAttentionClaudeAndPi; assertion: P9_SCHEMAS")
	for _, schema := range []string{"pi", "claude-code", "sdk-cli"} {
		t.Run(schema, func(t *testing.T) {
			path, id := attentionFixture(t, schema)
			r, e := buildAttention(path, []string{id}, "implementation")
			if e != nil {
				t.Fatalf("P9_SCHEMAS FAIL: %v", e)
			}
			if r.Page.Schema != schema || r.Page.Rooms[0].Metrics.Commands != 40 || r.Page.Rooms[0].Metrics.Edits != 0 || r.Page.Rooms[0].Result.Status != "match" {
				t.Fatalf("P9_SCHEMAS FAIL: %+v", r.Page)
			}
		})
	}
	if !t.Failed() {
		t.Log("P9_SCHEMAS PASS")
	}
}

func TestAttentionLimits(t *testing.T) {
	t.Log("procedure: TestAttentionLimits; assertion: P6_CLI")
	_, execute := setupNotebook(t)
	path, id := attentionFixture(t, "claude-code")
	for _, args := range [][]string{{}, {"--agent", "missing"}, {"--agent", id, "--agent", id}, {"--agent", id, "--format", "xml"}} {
		out, e := execute(append([]string{"transcript", "attention", path}, args...)...)
		if e == nil || strings.Contains(out, `"snapshot":`) {
			t.Fatalf("P6_CLI FAIL: invalid request published %v %s %v", args, out, e)
		}
	}
	// Existing inline Claude trees expose the child identity, but not its work.
	f, e := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.WriteString(`{"type":"assistant","uuid":"spawn","message":{"role":"assistant","content":[{"type":"tool_use","id":"child","name":"Task","input":{}}]}}` + "\n")
	f.Close()
	if e != nil {
		t.Fatal(e)
	}
	r, e := buildAttention(path, []string{"child"}, "implementation")
	if e != nil || r.Page.Rooms[0].Result.Status != "indeterminate" {
		t.Fatalf("P6_CLI FAIL: unavailable child evidence %v", e)
	}
	large := attentionRecord(t, 1, "assistant", fmt.Sprintf("%q", strings.Repeat("x", 1024*1024)))
	if _, _, _, e = collectAttention([]ledgerRecord{large}, "available", id, 100); e == nil {
		t.Fatal("P6_CLI FAIL: oversized message accepted")
	}
	var output bytes.Buffer
	if e = attentionJSON(&output, strings.Repeat("x", 200001)); e == nil || output.Len() != 0 {
		t.Fatal("P6_CLI FAIL: partial oversized publication")
	}
	t.Log("P6_CLI PASS")
}

func TestAttentionRetention(t *testing.T) {
	t.Log("procedure: TestAttentionRetention; assertion: P5_RETAINED")
	_, execute := setupNotebook(t)
	path, id := attentionFixture(t, "claude-code")
	out, e := execute("transcript", "attention", path, "--agent", id, "--task", "implementation", "--format", "json")
	if e != nil {
		t.Fatal(e)
	}
	var p attentionPage
	if e = json.Unmarshal([]byte(out), &p); e != nil {
		t.Fatal(e)
	}
	if e = os.Remove(path); e != nil {
		t.Fatal(e)
	}
	replay, e := execute("transcript", "attention", path, "--snapshot", p.Snapshot, "--format", "json")
	if e != nil || replay != out {
		t.Fatalf("P5_RETAINED FAIL: changed replay %v", e)
	}
	inspect, e := execute("transcript", "attention", "inspect", p.Snapshot, "--agent", id)
	if e != nil || !strings.Contains(inspect, "echo x") || !strings.Contains(inspect, p.Rooms[0].Window.FirstEvent) {
		t.Fatalf("P5_RETAINED FAIL: lost evidence: %v %s", e, inspect)
	}
	second, e := execute("transcript", "attention", "inspect", p.Snapshot, "--agent", id, "--page", "2", "--format", "json")
	var ep struct {
		Page, Pages int
		Evidence    []attentionEvidence
	}
	_ = json.Unmarshal([]byte(second), &ep)
	if e != nil || ep.Page != 2 || ep.Pages != 2 || len(ep.Evidence) != 20 || ep.Evidence[0].Ordinal != 21 || ep.Evidence[19].Ordinal != 40 {
		t.Fatalf("P5_RETAINED FAIL: incorrect inspection continuation %s %v", second, e)
	}
	if _, e = execute("transcript", "attention", "inspect", p.Snapshot, "--agent", id, "--page", "3"); e == nil {
		t.Fatal("P5_RETAINED FAIL: out of range page accepted")
	}
	if _, e = execute("transcript", "attention", path, "--snapshot", p.Snapshot, "--task", "research"); e == nil {
		t.Fatal("P5_RETAINED FAIL: scope mismatch accepted")
	}
	if _, e = execute("transcript", "attention", path+"wrong", "--snapshot", p.Snapshot); e == nil {
		t.Fatal("P5_RETAINED FAIL: path mismatch accepted")
	}
	dir, e := captureCacheDir()
	if e != nil {
		t.Fatal(e)
	}
	file := filepath.Join(dir, p.Snapshot+".attention")
	original, e := os.ReadFile(file)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(file, []byte("{}"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = loadAttention(p.Snapshot); e == nil {
		t.Fatal("P5_RETAINED FAIL: corrupt cache accepted")
	}
	if e = os.WriteFile(file, original, 0600); e != nil {
		t.Fatal(e)
	}
	past := time.Now().Add(-25 * time.Hour)
	if e = os.Chtimes(file, past, past); e != nil {
		t.Fatal(e)
	}
	if _, e = loadAttention(p.Snapshot); e == nil {
		t.Fatal("P5_RETAINED FAIL: expired cache accepted")
	}
	cleanupTranscriptCaptures()
	if _, e = os.Stat(file); !os.IsNotExist(e) {
		t.Fatal("P5_RETAINED FAIL: expired cache not pruned")
	}
	t.Log("P5_RETAINED PASS")
}

func TestAttentionPreservesRoomOrder(t *testing.T) {
	t.Log("procedure: TestAttentionPreservesRoomOrder; assertion: P8_ORDER")
	path := reviewFixture(t)
	before, e := buildReviewPage(path, "awaiting-return", "observed-recent", "", 20, "")
	if e != nil {
		t.Fatal(e)
	}
	ids := []string{"D", "B", "A"}
	r, e := buildAttention(path, ids, "implementation")
	if e != nil {
		t.Fatal(e)
	}
	for i, id := range ids {
		if r.Page.Rooms[i].ID != id {
			t.Fatal("P8_ORDER FAIL: explicit list reordered")
		}
	}
	after, e := buildReviewPage(path, "awaiting-return", "observed-recent", "", 20, "")
	if e != nil {
		t.Fatal(e)
	}
	a, _ := json.Marshal(before.Rows)
	b, _ := json.Marshal(after.Rows)
	if string(a) != string(b) {
		t.Fatal("P8_ORDER FAIL: awaiting-return membership/order changed")
	}
	// A has a parent return and is intentionally still evaluable.
	if r.Page.Rooms[2].ID != "A" {
		t.Fatal("P8_ORDER FAIL: returned room omitted")
	}
	t.Log("P8_ORDER PASS")
}

func TestAttentionNotebookReadOnly(t *testing.T) {
	t.Log("procedure: TestAttentionNotebookReadOnly; assertion: P4A_READONLY")
	nb, execute := setupNotebook(t)
	t.Setenv("NN_ATTENTION_TEST_NOTEBOOK", nb)
	_, e := execute("new", "--title", "Isolation sentinel", "--type", "observation", "--content", "```nn-rule\nviolation(kept, \"existing rule\").\n```\n```nn-attention\nviolation(leak, \"must stay isolated\").\n```", "--no-edit", "--no-suggest")
	if e != nil {
		t.Fatal(e)
	}
	digestNotebook := func() string {
		files, e := filepath.Glob(filepath.Join(nb, "*"))
		if e != nil {
			t.Fatal(e)
		}
		data := map[string]string{}
		for _, file := range files {
			info, e := os.Stat(file)
			if e != nil {
				t.Fatal(e)
			}
			if info.IsDir() {
				continue
			}
			b, e := os.ReadFile(file)
			if e != nil {
				t.Fatal(e)
			}
			data[file] = string(b)
		}
		b, _ := json.Marshal(data)
		return captureHash(b)
	}
	beforeFiles := digestNotebook()
	notes, e := execute("list", "--json")
	if e != nil {
		t.Fatal(e)
	}
	path, id := attentionFixture(t, "pi")
	if _, e = execute("transcript", "attention", path, "--agent", id, "--task", "implementation"); e != nil {
		t.Fatal(e)
	}
	after, e := execute("list", "--json")
	if e != nil || after != notes || digestNotebook() != beforeFiles {
		t.Fatalf("P4A_READONLY FAIL: notebook changed %v", e)
	}
	t.Log("P4A_READONLY PASS")
}

func TestAttentionNotebookRules(t *testing.T) {
	t.Log("procedure: TestAttentionNotebookRules; assertion: P4B_RULES")
	_, execute := setupNotebook(t)
	_, e := execute("new", "--title", "Rule sentinel", "--type", "observation", "--content", "```nn-rule\nviolation(kept, \"existing rule\").\n```\n```nn-attention\nviolation(leak, \"must stay isolated\").\n```", "--no-edit", "--no-suggest")
	if e != nil {
		t.Fatal(e)
	}
	before, e := execute("rules", "query", "violation")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(before, "kept") || strings.Contains(before, "leak") {
		t.Fatalf("P4B_RULES FAIL: fence isolation: %s", before)
	}
	path, id := attentionFixture(t, "pi")
	if _, e = execute("transcript", "attention", path, "--agent", id, "--task", "implementation"); e != nil {
		t.Fatal(e)
	}
	after, e := execute("rules", "query", "violation")
	if e != nil || after != before {
		t.Fatalf("P4B_RULES FAIL: notebook rules changed %v", e)
	}
	t.Log("P4B_RULES PASS")
}
