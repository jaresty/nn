package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/jaresty/nn/internal/note"
	"github.com/spf13/cobra"
)

func newReadCmd(state *rootState) *cobra.Command {
	var linesFlag string
	var limitFlag int

	cmd := &cobra.Command{
		Use:   "read <file>",
		Short: "Read a file with line numbers and append related notes from BM25 search",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read %s: %w", path, err)
			}

			allLines := strings.Split(string(data), "\n")
			// Remove trailing empty element from trailing newline.
			if len(allLines) > 0 && allLines[len(allLines)-1] == "" {
				allLines = allLines[:len(allLines)-1]
			}

			start, end := 1, len(allLines)
			if linesFlag != "" {
				parts := strings.SplitN(linesFlag, "-", 2)
				if len(parts) != 2 {
					return fmt.Errorf("--lines must be N-M (e.g. 10-20)")
				}
				s, err1 := strconv.Atoi(parts[0])
				e, err2 := strconv.Atoi(parts[1])
				if err1 != nil || err2 != nil || s < 1 || e < s {
					return fmt.Errorf("--lines: invalid range %q", linesFlag)
				}
				start, end = s, e
				if end > len(allLines) {
					end = len(allLines)
				}
				if start > len(allLines) {
					start = len(allLines) + 1
				}
			}

			shown := allLines[start-1 : end]
			if limitFlag > 0 && limitFlag < len(shown) {
				shown = shown[:limitFlag]
				end = start + limitFlag - 1
			}

			w := outWriter(cmd)
			for i, line := range shown {
				fmt.Fprintf(w, "%d\t%s\n", start+i, line)
			}

			// BM25 search on shown content.
			query := strings.Join(shown, " ")
			if strings.TrimSpace(query) == "" {
				return nil
			}

			notes, err := state.backend.List()
			if err != nil {
				return err
			}

			allInbound := make(map[string][]string)
			for _, n := range notes {
				for _, lnk := range n.Links {
					allInbound[lnk.TargetID] = append(allInbound[lnk.TargetID], lnk.Annotation)
				}
			}

			scores := note.BM25Scores(notes, query+" "+gitContextQuery(), allInbound)

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

			fmt.Fprintln(w, "\n## Related notes")
			fmt.Fprintln(w, "Resolve each related note before the next action — run `nn show <id>` to open, or write `skip-related: <id> [<id> ...] — <reason>` to dismiss. Continuing without resolving is a protocol violation.")
			for _, m := range matches {
				fmt.Fprintf(w, "- %s — %s %s\n", m.n.ID, m.n.Title, scoreLabel(m.score))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&linesFlag, "lines", "", "Line range to show (e.g. 10-20)")
	cmd.Flags().IntVar(&limitFlag, "limit", 0, "Maximum number of lines to show")
	return cmd
}

func scoreLabel(score float64) string {
	if score >= 1.0 {
		return "[likely relevant]"
	}
	return "[possibly relevant]"
}
