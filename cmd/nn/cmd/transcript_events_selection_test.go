package cmd

import (
	"encoding/json"
	"testing"
)

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
