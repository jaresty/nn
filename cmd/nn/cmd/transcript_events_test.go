package cmd

import (
	"encoding/json"
	"testing"
)

func TestTranscriptEventsCommand(t *testing.T) {
	session, _ := nativeToolResultFixture(t, true)
	_, execute := setupNotebook(t)
	out, err := execute("transcript", "events", session, "AAA", "--json")
	if err != nil {
		t.Fatalf("ASSERT_LEDGER_COMMAND: %v", err)
	}
	var p struct {
		Events []map[string]json.RawMessage `json:"events"`
		Detail string                       `json:"detail_status"`
	}
	if err := json.Unmarshal([]byte(out), &p); err != nil {
		t.Fatalf("ASSERT_LEDGER_COMMAND: %v", err)
	}
	if p.Detail != "available" || len(p.Events) != 3 {
		t.Fatalf("ASSERT_LEDGER_SELECTION: %s", out)
	}
	for _, e := range p.Events {
		if _, ok := e["payload"]; ok {
			t.Fatal("ASSERT_LEDGER_NO_PAYLOAD")
		}
	}
}
