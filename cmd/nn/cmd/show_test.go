package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: TestShowFreshnessFresh — note modified 1 day ago shows freshness: fresh with age and hint
func TestShowFreshnessFresh(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Fresh Note", note.TypeConcept)
	n.Modified = time.Now().UTC().Add(-24 * time.Hour)
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if !strings.Contains(out, "freshness: fresh") {
		t.Errorf("want freshness: fresh in output, got:\n%s", out)
	}
	if !strings.Contains(out, "likely current") {
		t.Errorf("want 'likely current' hint in output, got:\n%s", out)
	}
	if !strings.Contains(out, "ago") {
		t.Errorf("want age 'ago' in output, got:\n%s", out)
	}
}

// Assertion: TestShowFreshnessAging — note modified 7 days ago shows freshness: aging with age and hint
func TestShowFreshnessAging(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Aging Note", note.TypeConcept)
	n.Modified = time.Now().UTC().Add(-7 * 24 * time.Hour)
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if !strings.Contains(out, "freshness: aging") {
		t.Errorf("want freshness: aging in output, got:\n%s", out)
	}
	if !strings.Contains(out, "may need recheck") {
		t.Errorf("want 'may need recheck' hint in output, got:\n%s", out)
	}
	if !strings.Contains(out, "ago") {
		t.Errorf("want age 'ago' in output, got:\n%s", out)
	}
}

// Assertion: TestShowFreshnessStale — note modified 30 days ago shows freshness: stale with age and hint
func TestShowFreshnessStale(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Stale Note", note.TypeConcept)
	n.Modified = time.Now().UTC().Add(-30 * 24 * time.Hour)
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if !strings.Contains(out, "freshness: stale") {
		t.Errorf("want freshness: stale in output, got:\n%s", out)
	}
	if !strings.Contains(out, "content may be outdated") {
		t.Errorf("want 'content may be outdated' hint in output, got:\n%s", out)
	}
	if !strings.Contains(out, "ago") {
		t.Errorf("want age 'ago' in output, got:\n%s", out)
	}
}

func TestShowNote(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Show Me", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if !strings.Contains(out, "Show Me") {
		t.Errorf("output %q does not contain title 'Show Me'", out)
	}
}

func TestShowNoteNotFound(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("show", "99999999999999-0000")
	if err == nil {
		t.Fatal("nn show nonexistent: want error, got nil")
	}
}

// Assertion: TestShowProtocolNoDerivationBlock — plain nn show on a protocol note does NOT include ## Protocols block.
// The derivation block is only appended once in nn show --global output.
func TestShowProtocolNoDerivationBlock(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Protocol", note.TypeProtocol)
	n.Body = "Do the thing before acting."
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if strings.Contains(out, "## Protocols") {
		t.Errorf("expected no '## Protocols' derivation block in individual protocol note output; got:\n%s", out)
	}
}

// Assertion: TestShowNonProtocolNoDerivation — nn show on a concept note does NOT include ## Protocols block.
func TestShowNonProtocolNoDerivation(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	proto := newTestNoteForCLI(note.GenerateID(), "My Protocol", note.TypeProtocol)
	proto.Body = "Do the thing."
	concept := newTestNoteForCLI(note.GenerateID(), "My Concept", note.TypeConcept)
	concept.Body = "A concept about things."
	writeNoteFile(t, nbDir, proto)
	writeNoteFile(t, nbDir, concept)

	out, err := execute("show", concept.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if strings.Contains(out, "## Protocols") {
		t.Errorf("expected no '## Protocols' block for non-protocol note; got:\n%s", out)
	}
}

// Assertion: TestShowProtocolJSONNoDerivation — --json output does NOT include the derivation text.
func TestShowProtocolJSONNoDerivation(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Protocol", note.TypeProtocol)
	n.Body = "Do the thing before acting."
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID, "--json")
	if err != nil {
		t.Fatalf("nn show --json: %v", err)
	}
	if strings.Contains(out, "## Protocols") {
		t.Errorf("expected no derivation block in JSON output; got:\n%s", out)
	}
}

// Assertion: TestShowGlobalFlag — nn show --global prints all global protocol notes.
func TestShowGlobalFlag(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	p1 := newTestNoteForCLI(note.GenerateID(), "Protocol One", note.TypeProtocol)
	p2 := newTestNoteForCLI(note.GenerateID(), "Protocol Two", note.TypeProtocol)
	writeNoteFile(t, nbDir, p1)
	writeNoteFile(t, nbDir, p2)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "Protocol One") {
		t.Errorf("expected 'Protocol One' in output; got:\n%s", out)
	}
	if !strings.Contains(out, "Protocol Two") {
		t.Errorf("expected 'Protocol Two' in output; got:\n%s", out)
	}
}

// Assertion: TestShowGlobalEmpty — nn show --global with no notebook protocols still outputs virtual protocols.
func TestShowGlobalEmpty(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global with no protocols: %v", err)
	}
	if !strings.Contains(out, "virtual-nn-capture-discipline") {
		t.Errorf("expected virtual protocol in output even with empty notebook; got:\n%s", out)
	}
}

// Assertion: TestShowGlobalSeparator — multiple protocols are separated by ---.
func TestShowGlobalSeparator(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	p1 := newTestNoteForCLI(note.GenerateID(), "Protocol One", note.TypeProtocol)
	p2 := newTestNoteForCLI(note.GenerateID(), "Protocol Two", note.TypeProtocol)
	writeNoteFile(t, nbDir, p1)
	writeNoteFile(t, nbDir, p2)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "\n---\n") {
		t.Errorf("expected '---' separator between protocols; got:\n%s", out)
	}
}

func TestShowAppendsToAccessLog(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	n := newTestNoteForCLI(note.GenerateID(), "Access Me", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}

	logPath := filepath.Join(cfgDir, "access.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("access.log not created: %v", err)
	}
	if !strings.Contains(string(data), n.ID) {
		t.Errorf("access.log %q does not contain note ID %s", string(data), n.ID)
	}
}
