package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/jaresty/nn/internal/note"
	"github.com/spf13/cobra"
)

func newGrepCmd(state *rootState) *cobra.Command {
	var contextLines int
	var notesPerMatch int

	cmd := &cobra.Command{
		Use:   "grep <pattern> [path]",
		Short: "Search files for pattern and annotate matches with related nn notes",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pattern := args[0]
			re, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("invalid pattern %q: %w", pattern, err)
			}
			searchPath := "."
			if len(args) == 2 {
				searchPath = args[1]
			}

			// Collect lines from file(s).
			type fileLine struct {
				file    string
				lineNum int
				text    string
			}
			var allLines []fileLine

			info, err := os.Stat(searchPath)
			if err != nil {
				return fmt.Errorf("stat %s: %w", searchPath, err)
			}

			var files []string
			if info.IsDir() {
				if err := collectFiles(searchPath, &files); err != nil {
					return err
				}
			} else {
				files = []string{searchPath}
			}

			for _, f := range files {
				lines, err := readFileLines(f)
				if err != nil {
					continue
				}
				for i, line := range lines {
					allLines = append(allLines, fileLine{f, i + 1, line})
				}
			}

			// Find matches.
			type match struct {
				file    string
				lineNum int
				text    string
				context []string
			}
			var matches []match
			for idx, fl := range allLines {
				if !re.MatchString(fl.text) {
					continue
				}
				// Gather context lines.
				start := idx - contextLines
				if start < 0 {
					start = 0
				}
				end := idx + contextLines + 1
				if end > len(allLines) {
					end = len(allLines)
				}
				var ctx []string
				for _, cl := range allLines[start:end] {
					ctx = append(ctx, cl.text)
				}
				matches = append(matches, match{fl.file, fl.lineNum, fl.text, ctx})
			}

			if len(matches) == 0 {
				return nil
			}

			// Load notes for BM25.
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

			w := outWriter(cmd)
			k := notesPerMatch
			if k <= 0 {
				k = 2
			}

			for _, m := range matches {
				fmt.Fprintf(w, "%s:%d:%s\n", m.file, m.lineNum, m.text)

				query := strings.Join(m.context, " ")
				if strings.TrimSpace(query) == "" {
					continue
				}
				scores := note.BM25Scores(notes, query, allInbound)

				type scored struct {
					n     *note.Note
					score float64
				}
				var ranked []scored
				for _, n := range notes {
					if s := scores[n.ID]; s > 0 {
						ranked = append(ranked, scored{n, s})
					}
				}
				sort.Slice(ranked, func(i, j int) bool {
					return ranked[i].score > ranked[j].score
				})
				if len(ranked) > k {
					ranked = ranked[:k]
				}
				for _, r := range ranked {
					fmt.Fprintf(w, "  → [[%s|%s]]\n", r.n.ID, r.n.Title)
				}
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&contextLines, "context", 3, "Number of surrounding lines to include in BM25 query")
	cmd.Flags().IntVar(&notesPerMatch, "notes-per-match", 2, "Maximum related notes to show per match")
	return cmd
}

func collectFiles(dir string, out *[]string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		path := dir + "/" + e.Name()
		if e.IsDir() {
			if err := collectFiles(path, out); err != nil {
				return err
			}
		} else {
			*out = append(*out, path)
		}
	}
	return nil
}

func readFileLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}
