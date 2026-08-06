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

			// Parse comma-separated ranges into selected line numbers (1-based, sorted, deduplicated).
			var selectedLines []int
			if linesFlag != "" {
				seen := make(map[int]bool)
				for _, seg := range strings.Split(linesFlag, ",") {
					seg = strings.TrimSpace(seg)
					parts := strings.SplitN(seg, "-", 2)
					if len(parts) != 2 {
						return fmt.Errorf("--lines: each range must be N-M (e.g. 10-20), got %q", seg)
					}
					s, err1 := strconv.Atoi(parts[0])
					e, err2 := strconv.Atoi(parts[1])
					if err1 != nil || err2 != nil || s < 1 || e < s {
						return fmt.Errorf("--lines: invalid range %q", seg)
					}
					if s > len(allLines) {
						continue
					}
					if e > len(allLines) {
						e = len(allLines)
					}
					for i := s; i <= e; i++ {
						if !seen[i] {
							seen[i] = true
							selectedLines = append(selectedLines, i)
						}
					}
				}
				sort.Ints(selectedLines)
			} else {
				for i := 1; i <= len(allLines); i++ {
					selectedLines = append(selectedLines, i)
				}
			}

			if limitFlag > 0 && limitFlag < len(selectedLines) {
				selectedLines = selectedLines[:limitFlag]
			}

			var shown []string
			for _, ln := range selectedLines {
				shown = append(shown, allLines[ln-1])
			}

			w := outWriter(cmd)
			for i, line := range shown {
				fmt.Fprintf(w, "%d\t%s\n", selectedLines[i], line)
			}

			sessionReads := loadSessionReads(resolveCfgDir())

			// BM25 search on shown content.
			query := strings.Join(shown, " ")
			if strings.TrimSpace(query) == "" || state.backend == nil {
				return nil
			}

			notes, err := state.backend.List()
			if err != nil {
				return err
			}

			scores := RankedByQuery(notes, notes, query, state.notebookDir)

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
			hasUnread := false
			for _, m := range matches {
				readMarker := ""
				if sessionReads[m.n.ID] {
					readMarker = " [read]"
				} else {
					hasUnread = true
				}
				fmt.Fprintf(w, "- %s — %s %s%s\n", m.n.ID, m.n.Title, scoreLabel(m.score), readMarker)
			}
			printResolveInstruction(w, hasUnread)
			return nil
		},
	}

	cmd.Flags().StringVar(&linesFlag, "lines", "", "Line range(s) to show — single (e.g. 10-20) or comma-delimited (e.g. 1-5,10-20)")
	cmd.Flags().IntVar(&limitFlag, "limit", 0, "Maximum number of lines to show")
	return cmd
}

func scoreLabel(score float64) string {
	if score >= 1.0 {
		return "[likely relevant]"
	}
	return "[possibly relevant]"
}
