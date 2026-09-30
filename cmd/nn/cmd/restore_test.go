package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func historicalRestoreFixture(t *testing.T) (string, func(...string) (string, error), *note.Note, string, string) {
	t.Helper()
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Original Title", note.TypeConcept)
	n.Status = note.StatusDraft
	n.Tags = []string{"old-tag"}
	n.Body = "historical body\n\n## Historical detail\nkept as body"
	n.Links = []note.Link{{TargetID: "target-old", Type: "supports", Annotation: "historical link"}}
	writeNoteFile(t, nbDir, n)
	commitNoteFile(t, nbDir, n)
	revision := gitOutput(t, nbDir, "rev-parse", "HEAD")

	oldFilename := n.Filename()
	historicalBytes := gitOutput(t, nbDir, "show", revision+":"+oldFilename)
	historicalParsed, err := note.Parse([]byte(historicalBytes))
	if err != nil {
		t.Fatal(err)
	}
	n.Title = "Current Renamed Title"
	n.Status = note.StatusPermanent
	n.Tags = []string{"current-tag"}
	n.Body = "current body"
	n.Links = []note.Link{{TargetID: "target-current", Type: "refines", Annotation: "current link"}}
	n.Modified = n.Modified.Add(time.Hour)
	newFilename := n.Filename()
	if err := os.Remove(filepath.Join(nbDir, oldFilename)); err != nil {
		t.Fatal(err)
	}
	writeNoteFile(t, nbDir, n)
	gitOutput(t, nbDir, "add", "-A")
	gitOutput(t, nbDir, "commit", "-m", "test: rename and update note")
	if oldFilename == newFilename {
		t.Fatal("fixture did not rename note")
	}
	return nbDir, execute, n, revision, historicalParsed.Body
}

func TestRestoreHistoricalBodyPreservesCurrentNote(t *testing.T) {
	const assertion = "ASSERT_RESTORE_HISTORICAL_BODY_PRESERVES_CURRENT_NOTE"
	nbDir, execute, current, revision, historicalBody := historicalRestoreFixture(t)
	beforeCommits := gitCommitCount(t, nbDir)

	out, err := execute("restore", current.ID, "--from", revision, "--body", "--since", sinceFor(current))
	if err != nil {
		t.Fatalf("%s: %v", assertion, err)
	}
	if !strings.Contains(out, "restored "+current.ID) {
		t.Fatalf("%s output: %s", assertion, out)
	}
	data, err := os.ReadFile(filepath.Join(nbDir, current.Filename()))
	if err != nil {
		t.Fatal(err)
	}
	got, err := note.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Body != historicalBody {
		t.Errorf("%s body=%q want parsed historical body=%q", assertion, got.Body, historicalBody)
	}
	if got.Title != current.Title || got.Status != current.Status || strings.Join(got.Tags, ",") != "current-tag" {
		t.Errorf("%s metadata changed: %#v", assertion, got)
	}
	if len(got.Links) != 1 || got.Links[0].TargetID != "target-current" {
		t.Errorf("%s links changed: %#v", assertion, got.Links)
	}
	if gitCommitCount(t, nbDir) != beforeCommits+1 {
		t.Errorf("%s did not create exactly one commit", assertion)
	}
	t.Log(assertion + ": pass")
}

func TestRestoreDryRunDoesNotWrite(t *testing.T) {
	const assertion = "ASSERT_RESTORE_DRY_RUN_DOES_NOT_WRITE"
	nbDir, execute, current, revision, _ := historicalRestoreFixture(t)
	path := filepath.Join(nbDir, current.Filename())
	before, _ := os.ReadFile(path)
	head := gitOutput(t, nbDir, "rev-parse", "HEAD")

	out, err := execute("restore", current.ID, "--from", revision, "--body", "--dry-run")
	if err != nil {
		t.Fatalf("%s: %v", assertion, err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(before) || gitOutput(t, nbDir, "rev-parse", "HEAD") != head {
		t.Fatalf("%s mutated notebook", assertion)
	}
	if !strings.Contains(out, "--- current") || !strings.Contains(out, "+historical body") {
		t.Fatalf("%s missing body diff: %s", assertion, out)
	}
	t.Log(assertion + ": pass")
}

func TestRestoreRejectsInvalidInputs(t *testing.T) {
	const assertion = "ASSERT_RESTORE_REJECTS_INVALID_INPUTS"
	_, execute, current, revision, _ := historicalRestoreFixture(t)
	cases := map[string]struct {
		args []string
		want string
	}{
		"missing-body":     {[]string{"restore", current.ID, "--from", revision, "--since", sinceFor(current)}, "--body is required"},
		"missing-from":     {[]string{"restore", current.ID, "--body", "--since", sinceFor(current)}, "--from is required"},
		"invalid-revision": {[]string{"restore", current.ID, "--from", "not-a-revision", "--body", "--since", sinceFor(current)}, "invalid revision"},
		"stale-since":      {[]string{"restore", current.ID, "--from", revision, "--body", "--since", current.Modified.Add(-time.Second).Format(time.RFC3339Nano)}, "note was modified since"},
		"missing-since":    {[]string{"restore", current.ID, "--from", revision, "--body"}, "--since is required"},
	}
	for name, tc := range cases {
		_, err := execute(tc.args...)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s %s: error=%v, want %q", assertion, name, err, tc.want)
		}
	}
	if t.Failed() {
		return
	}
	t.Log(assertion + ": pass")
}

func TestHistoricalNoteAtRevisionRejectsMissingAndAmbiguous(t *testing.T) {
	const assertion = "ASSERT_RESTORE_HISTORICAL_ID_LOOKUP_IS_UNIQUE"
	nbDir, _, current, revision, _ := historicalRestoreFixture(t)
	if _, _, err := historicalNoteAtRevision(nbDir, revision, note.GenerateID()); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("%s missing: %v", assertion, err)
	}

	data, err := exec.Command("git", "-C", nbDir, "show", revision+":"+current.ID+"-original-title.md").Output()
	if err != nil {
		t.Fatal(err)
	}
	duplicate := current.ID + "-duplicate.md"
	if err := os.WriteFile(filepath.Join(nbDir, duplicate), data, 0o644); err != nil {
		t.Fatal(err)
	}
	gitOutput(t, nbDir, "add", duplicate)
	gitOutput(t, nbDir, "commit", "-m", "test: add duplicate historical id")
	ambiguousRevision := gitOutput(t, nbDir, "rev-parse", "HEAD")
	if _, _, err := historicalNoteAtRevision(nbDir, ambiguousRevision, current.ID); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("%s ambiguous: %v", assertion, err)
	}
	if t.Failed() {
		return
	}
	t.Log(assertion + ": pass")
}
