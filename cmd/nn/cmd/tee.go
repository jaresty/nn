package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/jaresty/nn/internal/note"
	"github.com/spf13/cobra"
)

const teeSearchLimit = 4096

func newTeeCmd(state *rootState) *cobra.Command {
	return &cobra.Command{
		Use:   "tee",
		Short: "Pass stdin to stdout unchanged; print BM25-matched related notes to stderr",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTee(os.Stdin, os.Stdout, os.Stderr, state, loadSessionReads(resolveCfgDir()))
		},
	}
}

func runTee(stdin io.Reader, stdout io.Writer, stderr io.Writer, state *rootState, sessionReads map[string]bool) error {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("tee: read stdin: %w", err)
	}

	if _, err := stdout.Write(data); err != nil {
		return fmt.Errorf("tee: write stdout: %w", err)
	}

	if state == nil || state.backend == nil {
		return nil
	}

	query := string(data)
	if len(query) > teeSearchLimit {
		query = query[:teeSearchLimit]
	}
	if strings.TrimSpace(query) == "" {
		return nil
	}

	notes, err := state.backend.List()
	if err != nil {
		return nil
	}

	allInbound := make(map[string][]string)
	for _, n := range notes {
		for _, lnk := range n.Links {
			allInbound[lnk.TargetID] = append(allInbound[lnk.TargetID], lnk.Annotation)
		}
	}

	scores := RankedByQuery(notes, allInbound, query, state.notebookDir)

	type scored struct {
		n     *note.Note
		score float64
	}
	var matches []scored
	for _, n := range notes {
		if s := scores[n.ID]; s > 0 {
			matches = append(matches, scored{n, s})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})
	if len(matches) > 5 {
		matches = matches[:5]
	}
	if len(matches) == 0 {
		return nil
	}

	fmt.Fprintln(stderr, "\n## Related notes")
	hasUnread := false
	for _, m := range matches {
		readMarker := ""
		if sessionReads[m.n.ID] {
			readMarker = " [read]"
		} else {
			hasUnread = true
		}
		fmt.Fprintf(stderr, "- %s — %s %s%s\n", m.n.ID, m.n.Title, scoreLabel(m.score), readMarker)
	}
	printResolveInstruction(stderr, hasUnread)
	return nil
}
