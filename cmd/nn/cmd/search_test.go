package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// ── nn list --search ──────────────────────────────────────────────────────────

func TestListSearchMatchesTitle(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Implicit Assumptions", note.TypeConcept)
	writeNoteFile(t, nbDir, n)
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Unrelated Note", note.TypeArgument))

	out, err := execute("list", "--search", "implicit")
	if err != nil {
		t.Fatalf("nn list --search: %v", err)
	}
	if !strings.Contains(out, "Implicit Assumptions") {
		t.Errorf("search output missing matching note: %q", out)
	}
	if strings.Contains(out, "Unrelated Note") {
		t.Errorf("search output contains non-matching note: %q", out)
	}
}

func TestListSearchMatchesBody(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Some Note", note.TypeConcept)
	n.Body = "This note discusses latent assumptions in design."
	writeNoteFile(t, nbDir, n)
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Other Note", note.TypeArgument))

	out, err := execute("list", "--search", "latent")
	if err != nil {
		t.Fatalf("nn list --search body: %v", err)
	}
	if !strings.Contains(out, "Some Note") {
		t.Errorf("search output missing body-matched note: %q", out)
	}
	if strings.Contains(out, "Other Note") {
		t.Errorf("search output contains non-matching note: %q", out)
	}
}

func TestListSearchCaseInsensitive(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Zettelkasten Method", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--search", "ZETTELKASTEN")
	if err != nil {
		t.Fatalf("nn list --search case: %v", err)
	}
	if !strings.Contains(out, "Zettelkasten Method") {
		t.Errorf("case-insensitive search failed: %q", out)
	}
}

func TestListSearchNoResults(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Some Note", note.TypeConcept))

	out, err := execute("list", "--search", "xyzzy")
	if err != nil {
		t.Fatalf("nn list --search no results: %v", err)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("search with no results should produce empty output, got: %q", out)
	}
}

// ── nn show title-prefix fallback ─────────────────────────────────────────────

func TestShowByTitlePrefix(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Ground Truth Principle", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", "ground")
	if err != nil {
		t.Fatalf("nn show by title prefix: %v", err)
	}
	if !strings.Contains(out, "Ground Truth Principle") {
		t.Errorf("show by prefix output missing title: %q", out)
	}
}

func TestShowByTitlePrefixAmbiguous(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Ground Truth", note.TypeConcept))
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Ground Rules", note.TypeArgument))

	_, err := execute("show", "ground")
	if err == nil {
		t.Fatal("nn show ambiguous prefix: want error, got nil")
	}
}

func TestShowExactIDStillWorks(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Exact Match", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show by exact ID: %v", err)
	}
	if !strings.Contains(out, "Exact Match") {
		t.Errorf("show by exact ID output missing title: %q", out)
	}
}
