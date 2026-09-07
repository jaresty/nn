package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranscriptEventsSDKFilenameIdentity(t *testing.T) {
	session := writeSDKCLIFixture(t, t.TempDir())
	dir := filepath.Join(strings.TrimSuffix(session, ".jsonl"), "subagents")
	a := filepath.Join(dir, "agent-aaa.jsonl")
	if err := os.Remove(a); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "agent-bbb.jsonl"), a); err != nil {
		t.Fatal(err)
	}
	_, execute := setupNotebook(t)
	if _, err := execute("transcript", "events", session, "aaa"); err == nil {
		t.Fatal("ASSERT_LEDGER_SDK_FILENAME: fail — another agent file accepted")
	}
	t.Log("ASSERT_LEDGER_SDK_FILENAME: pass")
}

func TestTranscriptEventsMissingMessage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pi.jsonl")
	writeTranscriptFile(t, path, `{"type":"session"}`+"\n"+`{"type":"message","message":null}`+"\n")
	_, execute := setupNotebook(t)
	events, p := ledgerAll(t, execute, path, "ROOT")
	if len(events) != 0 || p.DetailStatus != "unavailable" {
		t.Fatal("ASSERT_LEDGER_MISSING_MESSAGE: fail — no usable message reported available")
	}
	t.Log("ASSERT_LEDGER_MISSING_MESSAGE: pass")
}
