package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: TestReadCmd — nn read <file> outputs line-numbered file content.
func TestReadCmd(t *testing.T) {
	_, execute := setupNotebook(t)

	f := filepath.Join(t.TempDir(), "sample.go")
	content := "package main\n\nfunc hello() {}\n"
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("read", f)
	if err != nil {
		t.Fatalf("nn read: %v", err)
	}
	if !strings.Contains(out, "1\t") {
		t.Errorf("expected line-numbered output starting with '1\t'; got:\n%s", out)
	}
	if !strings.Contains(out, "package main") {
		t.Errorf("expected file content 'package main' in output; got:\n%s", out)
	}
}

// Assertion: TestReadCmdLines — nn read <file> --lines N-M outputs only lines N through M.
func TestReadCmdLines(t *testing.T) {
	_, execute := setupNotebook(t)

	f := filepath.Join(t.TempDir(), "sample.go")
	lines := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(f, []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("read", f, "--lines", "2-4")
	if err != nil {
		t.Fatalf("nn read --lines: %v", err)
	}
	if !strings.Contains(out, "line2") {
		t.Errorf("expected 'line2' in --lines 2-4 output; got:\n%s", out)
	}
	if !strings.Contains(out, "line4") {
		t.Errorf("expected 'line4' in --lines 2-4 output; got:\n%s", out)
	}
	if strings.Contains(out, "line1") {
		t.Errorf("expected 'line1' absent from --lines 2-4 output; got:\n%s", out)
	}
	if strings.Contains(out, "line5") {
		t.Errorf("expected 'line5' absent from --lines 2-4 output; got:\n%s", out)
	}
}

// Assertion: TestReadCmdLimit — nn read <file> --limit N caps output at N lines.
func TestReadCmdLimit(t *testing.T) {
	_, execute := setupNotebook(t)

	f := filepath.Join(t.TempDir(), "sample.go")
	var sb strings.Builder
	for i := 1; i <= 10; i++ {
		fmt.Fprintf(&sb, "line%d\n", i)
	}
	if err := os.WriteFile(f, []byte(sb.String()), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("read", f, "--limit", "3")
	if err != nil {
		t.Fatalf("nn read --limit: %v", err)
	}
	if !strings.Contains(out, "line3") {
		t.Errorf("expected 'line3' in --limit 3 output; got:\n%s", out)
	}
	if strings.Contains(out, "line4") {
		t.Errorf("expected 'line4' absent from --limit 3 output; got:\n%s", out)
	}
}

// Assertion: TestReadCmdRelatedNotes — nn read appends ## Related notes section with BM25 matches.
func TestReadCmdRelatedNotes(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "BM25 indexing and scoring", note.TypeConcept)
	n.Status = note.StatusReviewed
	n.Body = "BM25 is a ranking function used in information retrieval."
	writeNoteFile(t, nbDir, n)

	f := filepath.Join(t.TempDir(), "search.go")
	content := "// BM25 scoring implementation\npackage search\n"
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("read", f)
	if err != nil {
		t.Fatalf("nn read: %v", err)
	}
	if !strings.Contains(out, "## Related notes") {
		t.Errorf("expected '## Related notes' section in output; got:\n%s", out)
	}
	if !strings.Contains(out, "BM25 indexing and scoring") {
		t.Errorf("expected related note title 'BM25 indexing and scoring' in output; got:\n%s", out)
	}
}

// Assertion: TestReadCmdRelatedNotesInstruction — nn read appends nn show instruction after related notes.
func TestReadCmdRelatedNotesInstruction(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "BM25 indexing and scoring", note.TypeConcept)
	n.Status = note.StatusReviewed
	n.Body = "BM25 is a ranking function used in information retrieval."
	writeNoteFile(t, nbDir, n)

	f := filepath.Join(t.TempDir(), "search.go")
	content := "// BM25 scoring implementation\npackage search\n"
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("read", f)
	if err != nil {
		t.Fatalf("nn read: %v", err)
	}
	if !strings.Contains(out, "nn show") {
		t.Errorf("expected nn show instruction in related notes output; got:\n%s", out)
	}
	if !strings.Contains(out, "skip-related:") {
		t.Errorf("expected skip-related: in related notes output; got:\n%s", out)
	}
}

// Assertion: TestReadCmdRelatedNotesGateHeader — nn read related notes header uses gate language.
func TestReadCmdRelatedNotesGateHeader(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "BM25 indexing and scoring", note.TypeConcept)
	n.Status = note.StatusReviewed
	n.Body = "BM25 is a ranking function used in information retrieval."
	writeNoteFile(t, nbDir, n)

	f := filepath.Join(t.TempDir(), "search.go")
	content := "// BM25 scoring implementation\npackage search\n"
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("read", f)
	if err != nil {
		t.Fatalf("nn read: %v", err)
	}
	if !strings.Contains(out, "Resolve each related note before the next action") {
		t.Errorf("expected gate language 'Resolve each related note before the next action' in related notes header; got:\n%s", out)
	}
}

// Assertion: TestReadCmdRelatedNotesLabel — nn read appends score-derived label per related note.
func TestReadCmdRelatedNotesLabel(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "BM25 indexing and scoring", note.TypeConcept)
	n.Status = note.StatusReviewed
	n.Body = "BM25 is a ranking function used in information retrieval."
	writeNoteFile(t, nbDir, n)

	f := filepath.Join(t.TempDir(), "search.go")
	content := "// BM25 scoring implementation\npackage search\n"
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("read", f)
	if err != nil {
		t.Fatalf("nn read: %v", err)
	}
	if !strings.Contains(out, "[likely relevant]") && !strings.Contains(out, "[possibly relevant]") {
		t.Errorf("expected score-derived label ([likely relevant] or [possibly relevant]) in related notes output; got:\n%s", out)
	}
}

// Assertion: TestReadCmdCLIReference — nn read appears in CLI reference virtual protocol.
func TestReadCmdCLIReference(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("nn show virtual-nn-cli-reference: %v", err)
	}
	if !strings.Contains(out, "nn read") {
		t.Errorf("expected 'nn read' in CLI reference; got:\n%s", out)
	}
	if !strings.Contains(out, "--lines") {
		t.Errorf("expected '--lines' in CLI reference nn read entry; got:\n%s", out)
	}
}

// Assertion: TestReadCmdLinesStartBeyondEOF — nn read --lines N-M does not panic when N exceeds file length.
func TestReadCmdLinesStartBeyondEOF(t *testing.T) {
	_, execute := setupNotebook(t)

	f := filepath.Join(t.TempDir(), "short.go")
	if err := os.WriteFile(f, []byte("line1\nline2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("read", f, "--lines", "50-60")
	if err != nil {
		t.Fatalf("nn read --lines start beyond EOF: %v", err)
	}
	if strings.Contains(out, "line1") || strings.Contains(out, "line2") {
		t.Errorf("expected empty output when start exceeds file length; got:\n%s", out)
	}
}

// Assertion: TestReadCmdAllowList — nn read appears in capture-discipline allow-list.
func TestReadCmdAllowList(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "nn read") {
		t.Errorf("expected 'nn read' in capture-discipline allow-list; got:\n%s", out)
	}
}
