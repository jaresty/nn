package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestBulkUnlinkRemovesMultipleTargetsInOneCommit(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst1 := newTestNoteForCLI(note.GenerateID(), "Target One", note.TypeConcept)
	dst2 := newTestNoteForCLI(note.GenerateID(), "Target Two", note.TypeConcept)
	keep := newTestNoteForCLI(note.GenerateID(), "Keep", note.TypeConcept)
	src.Links = []note.Link{
		{TargetID: dst1.ID, Annotation: "first", Type: "supports"},
		{TargetID: dst1.ID, Annotation: "parallel", Type: "questions"},
		{TargetID: dst2.ID, Annotation: "second", Type: "extends"},
		{TargetID: keep.ID, Annotation: "unrelated", Type: "refines"},
	}
	for _, n := range []*note.Note{src, dst1, dst2, keep} {
		writeNoteFile(t, nbDir, n)
	}
	commitNoteFile(t, nbDir, src)
	before := gitCommitCount(t, nbDir)

	out, err := execute("bulk-unlink", src.ID, "--to", dst1.ID, "--to", dst2.ID)
	if err != nil {
		t.Fatalf("bulk-unlink: %v", err)
	}
	if !strings.Contains(out, "unlinked "+src.ID+" → 2 notes") {
		t.Fatalf("bulk-unlink output = %q", out)
	}
	after := gitCommitCount(t, nbDir)
	if after-before != 1 {
		t.Fatalf("bulk-unlink commits = %d, want 1", after-before)
	}
	shown, err := execute("show", src.ID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(shown, dst1.ID) || strings.Contains(shown, dst2.ID) {
		t.Fatalf("bulk-unlink retained requested targets:\n%s", shown)
	}
	if !strings.Contains(shown, keep.ID) {
		t.Fatalf("bulk-unlink removed unrelated target:\n%s", shown)
	}
}

func TestBulkUnlinkBroadcastTypePreservesParallelEdges(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst1 := newTestNoteForCLI(note.GenerateID(), "Target One", note.TypeConcept)
	dst2 := newTestNoteForCLI(note.GenerateID(), "Target Two", note.TypeConcept)
	src.Links = []note.Link{
		{TargetID: dst1.ID, Type: "supports", Annotation: "remove"},
		{TargetID: dst1.ID, Type: "questions", Annotation: "keep"},
		{TargetID: dst2.ID, Type: "supports", Annotation: "remove"},
		{TargetID: dst2.ID, Type: "extends", Annotation: "keep"},
	}
	for _, n := range []*note.Note{src, dst1, dst2} {
		writeNoteFile(t, nbDir, n)
	}

	if _, err := execute("bulk-unlink", src.ID, "--to", dst1.ID, "--to", dst2.ID, "--type", "supports"); err != nil {
		t.Fatalf("bulk-unlink broadcast type: %v", err)
	}
	shown, err := execute("show", src.ID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(shown, "remove") || !strings.Contains(shown, "[questions]") || !strings.Contains(shown, "[extends]") {
		t.Fatalf("bulk-unlink broadcast links:\n%s", shown)
	}
}

func TestBulkUnlinkPairsTypesByTarget(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst1 := newTestNoteForCLI(note.GenerateID(), "Target One", note.TypeConcept)
	dst2 := newTestNoteForCLI(note.GenerateID(), "Target Two", note.TypeConcept)
	src.Links = []note.Link{
		{TargetID: dst1.ID, Type: "supports"},
		{TargetID: dst1.ID, Type: "questions"},
		{TargetID: dst2.ID, Type: "supports"},
		{TargetID: dst2.ID, Type: "questions"},
	}
	for _, n := range []*note.Note{src, dst1, dst2} {
		writeNoteFile(t, nbDir, n)
	}

	if _, err := execute("bulk-unlink", src.ID,
		"--to", dst1.ID, "--type", "supports",
		"--to", dst2.ID, "--type", "questions",
	); err != nil {
		t.Fatalf("bulk-unlink paired types: %v", err)
	}
	shown, err := execute("show", src.ID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(shown, "[questions]") != 1 || strings.Count(shown, "[supports]") != 1 {
		t.Fatalf("bulk-unlink paired links:\n%s", shown)
	}
}

func TestBulkUnlinkIsDocumented(t *testing.T) {
	_, execute := setupNotebook(t)
	help, err := execute("bulk-unlink", "--help")
	if err != nil {
		t.Fatalf("bulk-unlink --help: %v", err)
	}
	for _, required := range []string{"bulk-unlink <from-id>", "single commit", "--to", "--type"} {
		if !strings.Contains(help, required) {
			t.Errorf("bulk-unlink help missing %q:\n%s", required, help)
		}
	}
	for _, path := range []string{"../../../skills/nn-guide/SKILL.md", "../../../skills/nn-workflow/SKILL.md"} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), "nn bulk-unlink <from-id>") {
			t.Errorf("%s missing bulk-unlink command", path)
		}
	}
}

func TestBulkUnlinkRejectsInvalidBatchWithoutMutation(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst1 := newTestNoteForCLI(note.GenerateID(), "Target One", note.TypeConcept)
	dst2 := newTestNoteForCLI(note.GenerateID(), "Target Two", note.TypeConcept)
	src.Links = []note.Link{{TargetID: dst1.ID, Type: "supports"}, {TargetID: dst2.ID, Type: "extends"}}
	for _, n := range []*note.Note{src, dst1, dst2} {
		writeNoteFile(t, nbDir, n)
	}

	if _, err := execute("bulk-unlink", src.ID); err == nil || !strings.Contains(err.Error(), "at least one --to is required") {
		t.Fatalf("bulk-unlink without --to error = %v", err)
	}
	if _, err := execute("bulk-unlink", src.ID,
		"--to", dst1.ID, "--to", dst2.ID,
		"--type", "supports", "--type", "extends", "--type", "questions",
	); err == nil || !strings.Contains(err.Error(), "counts must match") {
		t.Fatalf("bulk-unlink mismatched type error = %v", err)
	}
	shown, err := execute("show", src.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(shown, dst1.ID) || !strings.Contains(shown, dst2.ID) {
		t.Fatalf("invalid bulk-unlink mutated links:\n%s", shown)
	}
}
