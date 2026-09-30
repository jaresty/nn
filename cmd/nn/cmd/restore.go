package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newRestoreCmd(state *rootState) *cobra.Command {
	var revision, sinceStr string
	var bodyOnly, dryRun bool

	cmd := &cobra.Command{
		Use:   "restore <id-or-title>",
		Short: "Restore note content from Git history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if revision == "" {
				return fmt.Errorf("restore: --from is required")
			}
			if !bodyOnly {
				return fmt.Errorf("restore: --body is required; whole-note restore is not supported")
			}
			if !dryRun && sinceStr == "" {
				return fmt.Errorf("restore: --since is required unless --dry-run is used")
			}

			current, err := resolveNote(state, args[0])
			if err != nil {
				return fmt.Errorf("restore: %w", err)
			}
			historical, historicalPath, err := historicalNoteAtRevision(state.notebookDir, revision, current.ID)
			if err != nil {
				return fmt.Errorf("restore: %w", err)
			}
			// Parse retains the Markdown separator newline that Marshal writes before
			// Body. Remove exactly that structural prefix so the next Marshal+Parse
			// round trip yields the same parsed body as the historical note.
			restoredBody := strings.TrimPrefix(historical.Body, "\n")

			if dryRun {
				fmt.Fprint(outWriter(cmd), renderBodyRestoreDiff(current.Body, restoredBody, revision, historicalPath))
				return nil
			}

			since, err := parseRestoreSince(sinceStr)
			if err != nil {
				return err
			}
			current.Body = restoredBody
			current.Modified = updateNowFn()
			warnIfLarge(cmd, current.Body)
			if err := state.backend.Update(current, &since); err != nil {
				return fmt.Errorf("restore: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "restored %s body from %s:%s\nmodified: %s\n", current.ID, revision, historicalPath, current.Modified.Format(time.RFC3339Nano))
			return nil
		},
	}
	cmd.Flags().StringVar(&revision, "from", "", "Git revision containing the historical note")
	cmd.Flags().BoolVar(&bodyOnly, "body", false, "Restore only the parsed body; preserve current metadata and links")
	cmd.Flags().StringVar(&sinceStr, "since", "", "Reject restore if the current note changed after this RFC3339 timestamp")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show the proposed body diff without writing or committing")
	return cmd
}

func parseRestoreSince(value string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t, err = time.Parse(time.RFC3339, value)
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("restore: --since: invalid timestamp %q, want RFC3339", value)
	}
	return t, nil
}

func historicalNoteAtRevision(notebookDir, revision, id string) (*note.Note, string, error) {
	verify := exec.Command("git", "rev-parse", "--verify", revision+"^{commit}")
	verify.Dir = notebookDir
	if out, err := verify.CombinedOutput(); err != nil {
		return nil, "", fmt.Errorf("invalid revision %q: %s", revision, strings.TrimSpace(string(out)))
	}

	list := exec.Command("git", "ls-tree", "-r", "--name-only", revision)
	list.Dir = notebookDir
	out, err := list.Output()
	if err != nil {
		return nil, "", fmt.Errorf("list revision %q: %w", revision, err)
	}
	prefix := id + "-"
	var matches []string
	for _, path := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		base := filepath.Base(path)
		if strings.HasPrefix(base, prefix) && strings.HasSuffix(base, ".md") {
			matches = append(matches, path)
		}
	}
	if len(matches) == 0 {
		return nil, "", fmt.Errorf("note %s not found at revision %s", id, revision)
	}
	if len(matches) > 1 {
		return nil, "", fmt.Errorf("note %s is ambiguous at revision %s (%d files)", id, revision, len(matches))
	}

	show := exec.Command("git", "show", revision+":"+matches[0])
	show.Dir = notebookDir
	data, err := show.Output()
	if err != nil {
		return nil, "", fmt.Errorf("read %s at revision %s: %w", matches[0], revision, err)
	}
	historical, err := note.Parse(data)
	if err != nil {
		return nil, "", fmt.Errorf("parse %s at revision %s: %w", matches[0], revision, err)
	}
	if historical.ID != id {
		return nil, "", fmt.Errorf("historical file %s contains note id %q, want %q", matches[0], historical.ID, id)
	}
	return historical, matches[0], nil
}

func renderBodyRestoreDiff(current, historical, revision, path string) string {
	var b strings.Builder
	fmt.Fprintln(&b, "--- current")
	fmt.Fprintf(&b, "+++ %s:%s\n", revision, path)
	for _, line := range strings.Split(current, "\n") {
		fmt.Fprintf(&b, "-%s\n", line)
	}
	for _, line := range strings.Split(historical, "\n") {
		fmt.Fprintf(&b, "+%s\n", line)
	}
	return b.String()
}
