package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranscriptEventsDescriptionSelector(t *testing.T) {
	const assertion = "ASSERT_TRANSCRIPT_EVENTS_DESCRIPTION_SELECTOR"
	_, execute := setupNotebook(t)
	session := filepath.Join(t.TempDir(), "pi-events-description.jsonl")
	writeTranscriptFile(t, session, strings.Join([]string{
		`{"type":"session","version":3,"id":"01a","cwd":"/x"}`,
		`{"type":"message","id":"call","message":{"role":"assistant","content":[{"type":"toolCall","id":"agent-call","name":"Agent","arguments":{"description":"Inspect exact room","subagent_type":"general-purpose"}}]}}`,
		`{"type":"message","id":"result","parentId":"call","message":{"role":"toolResult","toolCallId":"agent-call","toolName":"Agent","details":{"status":"completed","agentId":"agent-a","description":"Inspect exact room","subagentType":"general-purpose"}}}`,
		`{"type":"custom","customType":"subagents:record","id":"done","data":{"id":"agent-a","type":"general-purpose","status":"completed","result":"done"}}`,
	}, "\n")+"\n")

	byID, err := execute("transcript", "events", session, "agent-a", "--at", "launch")
	if err != nil {
		t.Fatalf("%s id control: %v", assertion, err)
	}
	byDescription, err := execute("transcript", "events", session, "--description", "Inspect exact room", "--at", "launch")
	if err != nil || byDescription != byID {
		t.Fatalf("%s exact equivalence: err=%v\nid=%s\ndescription=%s", assertion, err, byID, byDescription)
	}
	t.Log(assertion + " unique exact resolution: pass")
	t.Log(assertion + " output equivalence: pass")
	for name, args := range map[string][]string{
		"empty":      {"transcript", "events", session, "--description", ""},
		"missing":    {"transcript", "events", session, "--description", "Missing room"},
		"case":       {"transcript", "events", session, "--description", "inspect exact room"},
		"positional": {"transcript", "events", session, "agent-a", "--description", "Inspect exact room"},
		"agent":      {"transcript", "events", session, "--agent", "agent-a", "--description", "Inspect exact room"},
	} {
		if _, err := execute(args...); err == nil {
			t.Errorf("%s %s: accepted invalid selector", assertion, name)
		}
	}
	if _, err := execute("transcript", "events", descriptionFixture(t), "--description", "Same room"); err == nil {
		t.Errorf("%s ambiguous: accepted multiple exact matches", assertion)
	}
	if t.Failed() {
		return
	}
	t.Log(assertion + " invalid cardinality rejection: pass")
	t.Log(assertion + " selector conflict rejection: pass")
}

func TestTranscriptEventsExactEvent(t *testing.T) {
	session := writePiFixture(t, t.TempDir())
	_, execute := setupNotebook(t)
	all, before := ledgerAll(t, execute, session, "ROOT", "--payload")
	target := all[1]
	selected, after := ledgerAll(t, execute, session, "ROOT", "--payload", "--event", target["event_id"].(string))
	if len(selected) != 1 {
		t.Fatal("ASSERT_LEDGER_EVENT_SELECTION: fail — selected more than the requested event")
	}
	want, _ := json.Marshal(target)
	got, _ := json.Marshal(selected[0])
	if string(want) != string(got) || before.Snapshot == after.Snapshot || after.EventFilter != target["event_id"] {
		t.Fatal("ASSERT_LEDGER_EVENT_SELECTION: fail — event changed or filter not bound")
	}
	if _, err := execute("transcript", "events", session, "ROOT", "--event", "missing"); err == nil {
		t.Fatal("ASSERT_LEDGER_EVENT_SELECTION: fail — unknown event accepted")
	}
	if _, err := execute("transcript", "events", session, "ROOT", "--payload", "--event", target["event_id"].(string), "--snapshot", before.Snapshot); err == nil {
		t.Fatal("ASSERT_LEDGER_EVENT_SELECTION: fail — incompatible snapshot accepted")
	}
	t.Log("ASSERT_LEDGER_EVENT_SELECTION: pass")
}
