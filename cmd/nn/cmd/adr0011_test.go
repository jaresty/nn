package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// ── Phase 1: Title-as-ID in mutating commands ─────────────────────────────────

func TestUpdateByTitle(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Ergonomic Update Target", note.TypeConcept)
	n.Body = "original body"
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", "Ergonomic Update", "--content", "replaced via title", "--no-edit")
	if err != nil {
		t.Fatalf("nn update by title: %v", err)
	}
	out, _ := execute("show", n.ID)
	if !strings.Contains(out, "replaced via title") {
		t.Errorf("content not updated via title-as-ID:\n%s", out)
	}
}

func TestDeleteByTitle(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Delete By Title Target", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("delete", "Delete By Title", "--confirm")
	if err != nil {
		t.Fatalf("nn delete by title: %v", err)
	}
	if _, err := os.Stat(filepath.Join(nbDir, n.Filename())); !os.IsNotExist(err) {
		t.Error("file still exists after delete by title")
	}
}

func TestPromoteByTitle(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Promote By Title Target", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("promote", "Promote By Title", "--to", "reviewed")
	if err != nil {
		t.Fatalf("nn promote by title: %v", err)
	}
	out, _ := execute("show", n.ID)
	if !strings.Contains(out, "reviewed") {
		t.Errorf("status not updated via promote by title:\n%s", out)
	}
}

func TestLinkByTitle(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	from := newTestNoteForCLI(note.GenerateID(), "Link From By Title", note.TypeConcept)
	to := newTestNoteForCLI(note.GenerateID(), "Link To Target", note.TypeConcept)
	writeNoteFile(t, nbDir, from)
	writeNoteFile(t, nbDir, to)

	_, err := execute("link", "Link From By Title", to.ID,
		"--annotation", "test link", "--type", "refines")
	if err != nil {
		t.Fatalf("nn link by title: %v", err)
	}
	out, _ := execute("show", from.ID)
	if !strings.Contains(out, to.ID) {
		t.Errorf("link not created via title-as-ID:\n%s", out)
	}
}

func TestUnlinkByTitle(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	from := newTestNoteForCLI(note.GenerateID(), "Unlink From By Title", note.TypeConcept)
	to := newTestNoteForCLI(note.GenerateID(), "Unlink To Target", note.TypeConcept)
	from.Links = []note.Link{{TargetID: to.ID, Annotation: "test link", Type: "refines"}}
	writeNoteFile(t, nbDir, from)
	writeNoteFile(t, nbDir, to)

	_, err := execute("unlink", "Unlink From By Title", to.ID)
	if err != nil {
		t.Fatalf("nn unlink by title: %v", err)
	}
	out, _ := execute("show", from.ID)
	if strings.Contains(out, to.ID) {
		t.Errorf("link still present after unlink by title:\n%s", out)
	}
}

// ── Phase 2: nn update --stdin ────────────────────────────────────────────────

func TestUpdateStdin(t *testing.T) {
	_, cfgFile := setupNotebookWithCfg(t)
	nbDir := extractNbDirFromCfg(t, cfgFile)
	n := newTestNoteForCLI(note.GenerateID(), "Stdin Update Note", note.TypeConcept)
	data, _ := n.Marshal()
	os.WriteFile(filepath.Join(nbDir, n.Filename()), data, 0o644)

	root := NewRootCmdForTest(cfgFile)
	root.SetArgs([]string{"update", n.ID, "--stdin", "--no-edit"})
	root.SetIn(strings.NewReader("body from stdin"))
	var stdout strings.Builder
	root.SetOut(&stdout)
	if err := root.Execute(); err != nil {
		t.Fatalf("nn update --stdin: %v", err)
	}

	root2 := NewRootCmdForTest(cfgFile)
	root2.SetArgs([]string{"show", n.ID})
	var out strings.Builder
	root2.SetOut(&out)
	_ = root2.Execute()
	if !strings.Contains(out.String(), "body from stdin") {
		t.Errorf("stdin body not applied:\n%s", out.String())
	}
}

// extractNbDirFromCfg reads the notebook path from a config file for tests.
func extractNbDirFromCfg(t *testing.T, cfgFile string) string {
	t.Helper()
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	// Config contains: path = "/tmp/..."
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "path = ") {
			p := strings.TrimPrefix(line, "path = ")
			p = strings.Trim(p, `"`)
			return p
		}
	}
	t.Fatalf("path not found in config %s", cfgFile)
	return ""
}

// ── Phase 3: nn update --replace-section ─────────────────────────────────────

func TestUpdateReplaceSection(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Section Note", note.TypeConcept)
	n.Body = "## Why\n\nOld explanation.\n\n## How\n\nThe method."
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--replace-section", "Why",
		"--content", "New explanation.", "--no-edit")
	if err != nil {
		t.Fatalf("nn update --replace-section: %v", err)
	}
	out, _ := execute("show", n.ID)
	if !strings.Contains(out, "New explanation.") {
		t.Errorf("section content not replaced:\n%s", out)
	}
	if strings.Contains(out, "Old explanation.") {
		t.Errorf("old section content still present:\n%s", out)
	}
	if !strings.Contains(out, "The method.") {
		t.Errorf("other section content was removed:\n%s", out)
	}
}

func TestUpdateReplaceSectionNotFound(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Section Note", note.TypeConcept)
	n.Body = "## Why\n\nExplanation."
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--replace-section", "Nonexistent",
		"--content", "New content.", "--no-edit")
	if err == nil {
		t.Fatal("nn update --replace-section with missing heading: want error, got nil")
	}
}

// ── Phase 4: nn update --status ──────────────────────────────────────────────

func TestUpdateStatus(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Status Note", note.TypeConcept)
	n.Status = note.StatusReviewed
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--status", "draft", "--no-edit")
	if err != nil {
		t.Fatalf("nn update --status: %v", err)
	}
	out, _ := execute("show", n.ID)
	if !strings.Contains(out, "draft") {
		t.Errorf("status not updated to draft:\n%s", out)
	}
}

func TestUpdateStatusInvalid(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Status Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--status", "invalid-status", "--no-edit")
	if err == nil {
		t.Fatal("nn update --status invalid: want error, got nil")
	}
}

// ── Phase 5: nn update --tags-add / --tags-remove ────────────────────────────

func TestUpdateTagsAdd(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Tags Note", note.TypeConcept)
	n.Tags = []string{"existing"}
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--tags-add", "new-tag", "--no-edit")
	if err != nil {
		t.Fatalf("nn update --tags-add: %v", err)
	}
	out, _ := execute("list", "--tag", "new-tag", "--json")
	if !strings.Contains(out, n.ID) {
		t.Errorf("note not found by added tag:\n%s", out)
	}
	out2, _ := execute("list", "--tag", "existing", "--json")
	if !strings.Contains(out2, n.ID) {
		t.Errorf("existing tag removed after --tags-add:\n%s", out2)
	}
}

func TestUpdateTagsRemove(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Tags Note", note.TypeConcept)
	n.Tags = []string{"keep", "remove-me"}
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--tags-remove", "remove-me", "--no-edit")
	if err != nil {
		t.Fatalf("nn update --tags-remove: %v", err)
	}
	out, _ := execute("list", "--tag", "remove-me", "--json")
	if strings.Contains(out, n.ID) {
		t.Errorf("removed tag still applied:\n%s", out)
	}
	out2, _ := execute("list", "--tag", "keep", "--json")
	if !strings.Contains(out2, n.ID) {
		t.Errorf("kept tag was also removed:\n%s", out2)
	}
}

func TestUpdateTagsAddAndRemoveCompose(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Tags Compose Note", note.TypeConcept)
	n.Tags = []string{"inbox", "keep"}
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID,
		"--tags-add", "zettelkasten",
		"--tags-remove", "inbox",
		"--no-edit")
	if err != nil {
		t.Fatalf("nn update --tags-add --tags-remove: %v", err)
	}
	outAdded, _ := execute("list", "--tag", "zettelkasten", "--json")
	if !strings.Contains(outAdded, n.ID) {
		t.Errorf("added tag not applied:\n%s", outAdded)
	}
	outRemoved, _ := execute("list", "--tag", "inbox", "--json")
	if strings.Contains(outRemoved, n.ID) {
		t.Errorf("removed tag still applied:\n%s", outRemoved)
	}
	outKept, _ := execute("list", "--tag", "keep", "--json")
	if !strings.Contains(outKept, n.ID) {
		t.Errorf("kept tag was removed:\n%s", outKept)
	}
}
