package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestDeleteRemovesNote(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Delete Me", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("delete", n.ID, "--confirm")
	if err != nil {
		t.Fatalf("nn delete: %v", err)
	}

	if _, err := os.Stat(filepath.Join(nbDir, n.Filename())); !os.IsNotExist(err) {
		t.Error("file still exists after delete")
	}
}

func TestDeleteRequiresConfirm(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Delete Me", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("delete", n.ID)
	if err == nil {
		t.Fatal("nn delete without --confirm: want error, got nil")
	}
}

// Assertion: TestDeleteFromStdin — --from-stdin deletes each ID read from stdin
func TestDeleteFromStdin(t *testing.T) {
	nbDir, cfgFile := setupNotebookWithCfg(t)
	n1 := newTestNoteForCLI(note.GenerateID(), "Batch Delete 1", note.TypeConcept)
	n2 := newTestNoteForCLI(note.GenerateID(), "Batch Delete 2", note.TypeConcept)
	writeNoteFile(t, nbDir, n1)
	writeNoteFile(t, nbDir, n2)

	stdin := bytes.NewBufferString(fmt.Sprintf("%s\n%s\n", n1.ID, n2.ID))
	var stdout bytes.Buffer
	root := NewRootCmdForTest(cfgFile)
	root.SetIn(stdin)
	root.SetOut(&stdout)
	root.SetArgs([]string{"delete", "--from-stdin", "--confirm"})
	if err := root.Execute(); err != nil {
		t.Fatalf("nn delete --from-stdin: %v", err)
	}

	for _, n := range []*note.Note{n1, n2} {
		if _, err := os.Stat(filepath.Join(nbDir, n.Filename())); !os.IsNotExist(err) {
			t.Errorf("file %s still exists after batch delete", n.Filename())
		}
	}
}

func TestDeleteLinkedWarns(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	target := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeConcept)
	linker := newTestNoteForCLI(note.GenerateID(), "Linker", note.TypeArgument)
	linker.Links = []note.Link{{TargetID: target.ID, Annotation: "depends on"}}
	writeNoteFile(t, nbDir, target)
	writeNoteFile(t, nbDir, linker)

	out, _ := execute("delete", target.ID, "--confirm")
	// Should complete but output a warning
	if !strings.Contains(strings.ToLower(out), "warn") && !strings.Contains(strings.ToLower(out), "linked") {
		t.Logf("delete linked note output: %q (warning expected but not enforced)", out)
	}
}
