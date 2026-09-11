package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writePiReceiptTranscript(t *testing.T, dir, id string) string {
	t.Helper()
	path := filepath.Join(dir, id+".jsonl")
	if err := os.WriteFile(path, []byte("{\"type\":\"session\",\"version\":1,\"id\":\""+id+"\"}\n{\"type\":\"message\",\"content\":\"SECRET RAW BODY\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTranscriptReceiptCreatesBoundedExpiringNote(t *testing.T) {
	_, execute := setupNotebook(t)
	transcript := writePiReceiptTranscript(t, t.TempDir(), "session-123")
	fixed := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	orig := transcriptReceiptNow
	transcriptReceiptNow = func() time.Time { return fixed }
	defer func() { transcriptReceiptNow = orig }()

	out, err := execute("transcript", "receipt", transcript,
		"--assignment", "Map receipt implementation",
		"--disposition", "partially-adopted",
		"--parent-session", "parent-7",
		"--adopted", "Shared resolver is reusable",
		"--rejected", "Do not copy raw output",
		"--result", "commit abc123",
		"--verification", "tests passed")
	if err != nil {
		t.Fatalf("receipt: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 3 || !strings.HasPrefix(lines[0], "created ") || lines[1] != "session: session-123" || lines[2] != "expires: 2026-09-25" {
		t.Fatalf("receipt output:\n%s", out)
	}
	id := strings.TrimPrefix(lines[0], "created ")
	shown, err := execute("show", id)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"type: observation", "status: draft", "subagent-handoff", "transcript-receipt", "expires: 2026-09-25", "provider: pi", "session: session-123", "parent_session: parent-7", "disposition: partially-adopted", "## Adopted", "Shared resolver is reusable", "## Rejected or revised", "Do not copy raw output", "commit abc123", "tests passed", "nn transcript show session-123"} {
		if !strings.Contains(shown, want) {
			t.Errorf("receipt missing %q:\n%s", want, shown)
		}
	}
	if strings.Contains(shown, "SECRET RAW BODY") {
		t.Fatal("receipt copied raw transcript body")
	}
}

func TestTranscriptReceiptResolvesDiscoveredSessionID(t *testing.T) {
	_, execute := setupNotebook(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := filepath.Join(home, ".pi", "agent", "sessions", "project")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	writePiReceiptTranscript(t, root, "discovered-session")

	out, err := execute("transcript", "receipt", "discovered-session",
		"--assignment", "Resolve discovered transcript", "--disposition", "accepted")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "session: discovered-session") {
		t.Fatalf("receipt output:\n%s", out)
	}
}

func TestTranscriptReceiptValidatesRequiredFields(t *testing.T) {
	_, execute := setupNotebook(t)
	transcript := writePiReceiptTranscript(t, t.TempDir(), "session-validate")
	cases := [][]string{
		{"transcript", "receipt", transcript, "--disposition", "accepted"},
		{"transcript", "receipt", transcript, "--assignment", "task", "--disposition", "unknown"},
		{"transcript", "receipt", transcript, "--assignment", "task", "--disposition", "accepted", "--expires-in", "0s"},
		{"transcript", "receipt", transcript, "--assignment", "task", "--disposition", "accepted", "--link-to", "x"},
		{"transcript", "receipt", transcript, "--assignment", "task", "--disposition", "accepted", "--link-to", "missing-note", "--link-type", "follows", "--annotation", "missing target"},
		{"transcript", "receipt", filepath.Join(t.TempDir(), "missing.jsonl"), "--assignment", "task", "--disposition", "accepted"},
	}
	for _, args := range cases {
		if _, err := execute(args...); err == nil {
			t.Errorf("expected validation error for %v", args)
		}
	}
}

func TestTranscriptReceiptPersistsLinksInOneCommit(t *testing.T) {
	dir, execute := setupNotebook(t)
	transcript := writePiReceiptTranscript(t, t.TempDir(), "session-link")
	targetOut, err := execute("new", "--title", "Receipt target", "--type", "observation", "--no-edit", "--no-suggest")
	if err != nil {
		t.Fatal(err)
	}
	targetID := strings.TrimSpace(strings.TrimPrefix(targetOut, "created "))
	before := gitCommitCount(t, dir)
	out, err := execute("transcript", "receipt", transcript,
		"--assignment", "Integrate linked result", "--disposition", "accepted",
		"--link-to", targetID, "--link-type", "follows", "--annotation", "Records integration after the task")
	if err != nil {
		t.Fatal(err)
	}
	if after := gitCommitCount(t, dir); after != before+1 {
		t.Fatalf("receipt commits = %d, want %d", after-before, 1)
	}
	id := strings.TrimPrefix(strings.Split(strings.TrimSpace(out), "\n")[0], "created ")
	shown, err := execute("show", id)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(shown, "[["+targetID+"|Receipt target]] [follows]") {
		t.Fatalf("receipt link not persisted:\n%s", shown)
	}
}

func TestTranscriptReceiptProtocolIsNarrowlyTriggered(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"transcript integration receipt", "consequentially adopted", "Do not create a receipt for every subagent return", "nn transcript receipt"} {
		if !strings.Contains(out, want) {
			t.Errorf("global protocols missing %q", want)
		}
	}
	for _, path := range []string{"../../../skills/nn-transcript/SKILL.md", "../../../skills/nn-transcript/references/actions.md"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "integration receipt") {
			t.Errorf("%s lacks integration receipt guidance", path)
		}
	}
}
