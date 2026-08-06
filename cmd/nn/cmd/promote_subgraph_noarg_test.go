package cmd

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [3]: --subgraph does not require a positional argument
func TestPromoteSubgraphNoPositionalArg(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := newTestNoteForCLI(note.GenerateID(), "Root Model", note.TypeModel)
	root.Representation = "ontology"
	root.Status = note.StatusDraft
	writeNoteFile(t, nbDir, root)

	// Should not require a positional arg when --subgraph is provided
	_, err := execute("promote", "--subgraph", root.ID, "--to", "reviewed")
	if err != nil {
		t.Fatalf("nn promote --subgraph without positional arg: %v", err)
	}
}
