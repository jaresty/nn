package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

func TestListStaleReturnsAccessedUnactedNotes(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	n := newTestNoteForCLI(note.GenerateID(), "Stale Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	// Write an access.log entry 1 hour ago for this note.
	accessTime := time.Now().Add(-1 * time.Hour)
	logLine := fmt.Sprintf("%s show %s\n", accessTime.UTC().Format(time.RFC3339), n.ID)
	if err := os.WriteFile(filepath.Join(cfgDir, "access.log"), []byte(logLine), 0o644); err != nil {
		t.Fatal(err)
	}

	// Note has not been committed since access — no git commits in repo at all.
	out, err := execute("list", "--unactioned")
	if err != nil {
		t.Fatalf("nn list --stale: %v", err)
	}
	if !strings.Contains(out, n.ID) {
		t.Errorf("nn list --stale: expected note %s in output, got %q", n.ID, out)
	}
}

func TestListStaleExcludesRecentlyCommittedNotes(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	n := newTestNoteForCLI(note.GenerateID(), "Fresh Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	// Commit the note (simulate post-access action).
	commitNoteFile(t, nbDir, n)

	// Write an access.log entry 1 hour BEFORE the commit (note was committed after access).
	accessTime := time.Now().Add(-2 * time.Hour)
	logLine := fmt.Sprintf("%s show %s\n", accessTime.UTC().Format(time.RFC3339), n.ID)
	if err := os.WriteFile(filepath.Join(cfgDir, "access.log"), []byte(logLine), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("list", "--unactioned")
	if err != nil {
		t.Fatalf("nn list --stale: %v", err)
	}
	if strings.Contains(out, n.ID) {
		t.Errorf("nn list --stale: note %s should be excluded (committed after access), got %q", n.ID, out)
	}
}

// Assertion: TestListUnactionedAccepted — --unactioned accepted after rename
func TestListUnactionedAccepted(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("list", "--unactioned")
	if err != nil {
		t.Errorf("nn list --unactioned should be accepted, got: %v", err)
	}
}

// Assertion: TestListStaleRejected — --stale rejected after rename to --unactioned
func TestListStaleRejected(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("list", "--stale")
	if err == nil {
		t.Errorf("nn list --stale should be rejected after rename to --unactioned")
	}
}

// Assertion: TestListOlderThan — --older-than <days> filters notes not modified in N+ days
func TestListOlderThan(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	old := newTestNoteForCLI(note.GenerateID(), "Old Note", note.TypeConcept)
	old.Modified = time.Now().UTC().Add(-20 * 24 * time.Hour)
	writeNoteFile(t, nbDir, old)

	fresh := newTestNoteForCLI(note.GenerateID(), "Fresh Note", note.TypeConcept)
	fresh.Modified = time.Now().UTC().Add(-1 * time.Hour)
	writeNoteFile(t, nbDir, fresh)

	out, err := execute("list", "--older-than", "14")
	if err != nil {
		t.Fatalf("nn list --older-than: %v", err)
	}
	if !strings.Contains(out, old.ID) {
		t.Errorf("want old note %s in --older-than output, got:\n%s", old.ID, out)
	}
	if strings.Contains(out, fresh.ID) {
		t.Errorf("fresh note %s should be excluded from --older-than output", fresh.ID)
	}
}

// Assertion: TestListNoInboundIncludesDeadEnd — note with outbound but no inbound links appears
func TestListNoInboundIncludesDeadEnd(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	target := newTestNoteForCLI(note.GenerateID(), "Target Note", note.TypeConcept)
	writeNoteFile(t, nbDir, target)

	src := newTestNoteForCLI(note.GenerateID(), "Source Note", note.TypeConcept)
	src.Links = []note.Link{{TargetID: target.ID, Type: "extends", Annotation: "extends target"}}
	writeNoteFile(t, nbDir, src)

	out, err := execute("list", "--no-inbound")
	if err != nil {
		t.Fatalf("nn list --no-inbound: %v", err)
	}
	if !strings.Contains(out, src.ID) {
		t.Errorf("want dead-end note %s in --no-inbound output, got:\n%s", src.ID, out)
	}
}

// Assertion: TestListNoInboundExcludesLinkedTarget — note with inbound links excluded
func TestListNoInboundExcludesLinkedTarget(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	target := newTestNoteForCLI(note.GenerateID(), "Target Note", note.TypeConcept)
	writeNoteFile(t, nbDir, target)

	src := newTestNoteForCLI(note.GenerateID(), "Source Note", note.TypeConcept)
	src.Links = []note.Link{{TargetID: target.ID, Type: "extends", Annotation: "extends target"}}
	writeNoteFile(t, nbDir, src)

	out, err := execute("list", "--no-inbound")
	if err != nil {
		t.Fatalf("nn list --no-inbound: %v", err)
	}
	if strings.Contains(out, target.ID) {
		t.Errorf("target note %s has inbound links and should be excluded from --no-inbound output", target.ID)
	}
}
