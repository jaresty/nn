package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

// [24]: a known-schema session with an unresolved parent must NOT hard-fail;
// tree re-parents the orphan and still renders.
func TestTranscriptTreeOrphanDoesNotAbort(t *testing.T) {
	dir := t.TempDir()
	// pi session: a subagent whose parentId points at a non-existent record.
	session := filepath.Join(dir, "orphan.jsonl")
	writeTranscriptFile(t, session,
		`{"type":"session","version":3,"id":"01a","cwd":"/x"}`+"\n"+
			`{"type":"message","id":"m1","message":{"role":"assistant","content":[{"type":"toolCall","id":"call_1","name":"Agent","arguments":{}}]}}`+"\n"+
			`{"type":"custom","customType":"subagents:record","id":"c1","parentId":"m1","data":{"id":"good","status":"completed"}}`+"\n"+
			`{"type":"custom","customType":"subagents:record","id":"c2","parentId":"MISSING-RECORD","data":{"id":"orphan","status":"completed"}}`+"\n")
	_, execute := setupNotebook(t)

	// text tree must succeed (no abort) and show both agents.
	out, err := execute("transcript", "tree", session)
	if err != nil {
		t.Fatalf("tree should not abort on an orphan in a known schema: %v", err)
	}
	if !strings.Contains(out, "good") || !strings.Contains(out, "orphan") {
		t.Errorf("expected both good and orphan agents rendered:\n%s", out)
	}
}

// [25]: show ROOT on a pi session renders the main-thread events.
func TestTranscriptShowPiRoot(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "pi.jsonl")
	writeTranscriptFile(t, session,
		`{"type":"session","version":3,"id":"01a","cwd":"/x"}`+"\n"+
			`{"type":"message","id":"m1","message":{"role":"user","content":"Do the thing please."}}`+"\n"+
			`{"type":"message","id":"m2","message":{"role":"assistant","content":[{"type":"text","text":"Working on it."},{"type":"toolCall","id":"call_1","name":"bash","arguments":{"command":"ls"}}]}}`+"\n")
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "show", session, "ROOT")
	if err != nil {
		t.Fatalf("show ROOT: %v", err)
	}
	if strings.Contains(out, "no per-agent detail") {
		t.Errorf("show ROOT should render pi main-thread events, got:\n%s", out)
	}
	if !strings.Contains(out, "Do the thing") {
		t.Errorf("expected the user prompt in ROOT events:\n%s", out)
	}
	if !strings.Contains(out, "bash") {
		t.Errorf("expected the tool call in ROOT events:\n%s", out)
	}
}

// [26]: pi recipe resolves a spawn owner from ANY record hosting an Agent
// toolCall, not only the ROOT stream (nested spawns resolve).
func TestTranscriptPiNestedSpawnResolves(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "nested.jsonl")
	// m1 (ROOT) hosts Agent spawning child A (c1/data.id=A).
	// mA is a record attributed to A that itself hosts an Agent toolCall,
	// spawning grandchild B (c2, parentId=mA). B must resolve to parent A.
	writeTranscriptFile(t, session,
		`{"type":"session","version":3,"id":"01a","cwd":"/x"}`+"\n"+
			`{"type":"message","id":"m1","message":{"role":"assistant","content":[{"type":"toolCall","id":"call_1","name":"Agent","arguments":{}}]}}`+"\n"+
			`{"type":"custom","customType":"subagents:record","id":"c1","parentId":"m1","data":{"id":"A","status":"completed"}}`+"\n"+
			`{"type":"message","id":"mA","agentId":"A","message":{"role":"assistant","content":[{"type":"toolCall","id":"call_2","name":"Agent","arguments":{}}]}}`+"\n"+
			`{"type":"custom","customType":"subagents:record","id":"c2","parentId":"mA","data":{"id":"B","status":"completed"}}`+"\n")
	_, execute := setupNotebook(t)

	out, err := execute("transcript", "tree", session, "--json")
	if err != nil {
		t.Fatalf("tree: %v", err)
	}
	rows := parseTreeJSON(t, out)
	b, ok := rowByID(rows, "B")
	if !ok {
		t.Fatalf("expected grandchild B: %+v", rows)
	}
	// B's spawning record mA is owned by agent A → B.parent should be A, not dangling.
	if b.ParentID != "A" {
		t.Errorf("nested spawn: B parent should resolve to A, got %q", b.ParentID)
	}
}
