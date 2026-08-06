package cmd

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jaresty/nn/internal/ast"
	"github.com/jaresty/nn/internal/note"
	"github.com/spf13/cobra"
)

func newShufCmd(state *rootState) *cobra.Command {
	var count int
	var unit string

	cmd := &cobra.Command{
		Use:   "shuf [<path>...]",
		Short: "Sample random units from files (or stdin) and show BM25-matched notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShuf(cmd.InOrStdin(), args, cmd.OutOrStdout(), state, count, unit)
		},
	}
	cmd.Flags().IntVar(&count, "count", 5, "Number of units to sample")
	cmd.Flags().StringVar(&unit, "unit", "paragraphs", "Sampling unit: lines, paragraphs, or symbols")
	return cmd
}

func runShuf(stdin io.Reader, paths []string, out io.Writer, state *rootState, count int, unit string) error {
	units, err := collectUnits(stdin, paths, unit)
	if err != nil {
		return err
	}
	if len(units) == 0 {
		return nil
	}

	// Sample without replacement (or with replacement if count > len).
	indices := rand.Perm(len(units))
	if count < len(indices) {
		indices = indices[:count]
	}

	var notes []*note.Note
	allInbound := make(map[string][]string)
	if state != nil && state.backend != nil {
		notes, _ = state.backend.List()
		for _, n := range notes {
			for _, lnk := range n.Links {
				allInbound[lnk.TargetID] = append(allInbound[lnk.TargetID], lnk.Annotation)
			}
		}
	}

	for _, idx := range indices {
		u := units[idx]
		fmt.Fprintln(out, "---")
		fmt.Fprintln(out, u)

		if len(notes) > 0 && strings.TrimSpace(u) != "" {
			printShufRelated(out, notes, allInbound, u)
		}
	}
	return nil
}

func collectUnits(stdin io.Reader, paths []string, unit string) ([]string, error) {
	if len(paths) == 0 {
		if stdin == nil {
			return nil, nil
		}
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("shuf: read stdin: %w", err)
		}
		return splitUnits(string(data), unit, ""), nil
	}

	var all []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("shuf: stat %s: %w", p, err)
		}
		if info.IsDir() {
			if err := filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				data, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("shuf: read %s: %w", path, err)
				}
				all = append(all, splitUnits(string(data), unit, path)...)
				return nil
			}); err != nil {
				return nil, err
			}
		} else {
			data, err := os.ReadFile(p)
			if err != nil {
				return nil, fmt.Errorf("shuf: read %s: %w", p, err)
			}
			all = append(all, splitUnits(string(data), unit, p)...)
		}
	}
	return all, nil
}

func splitUnits(content, unit, filePath string) []string {
	switch unit {
	case "lines":
		return splitLines(content)
	case "symbols":
		if filePath != "" {
			if syms := extractSymbolBodies(filePath); len(syms) > 0 {
				return syms
			}
		}
		return splitParagraphs(content)
	default: // paragraphs
		return splitParagraphs(content)
	}
}

func splitLines(content string) []string {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func splitParagraphs(content string) []string {
	var paras []string
	var cur strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			if cur.Len() > 0 {
				paras = append(paras, strings.TrimSpace(cur.String()))
				cur.Reset()
			}
		} else {
			cur.WriteString(line)
			cur.WriteByte('\n')
		}
	}
	if cur.Len() > 0 {
		paras = append(paras, strings.TrimSpace(cur.String()))
	}
	return paras
}

func extractSymbolBodies(filePath string) []string {
	f, err := ast.Parse(filePath)
	if err != nil {
		return nil
	}
	var bodies []string
	for _, sym := range f.Symbols {
		if sym.Kind == "import" {
			continue
		}
		text := sym.Body
		if text == "" {
			text = sym.Signature
		}
		if strings.TrimSpace(text) != "" {
			bodies = append(bodies, text)
		}
	}
	return bodies
}

func printShufRelated(out io.Writer, notes []*note.Note, allInbound map[string][]string, query string) {
	scores := note.BM25Scores(notes, query, allInbound)
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
	sort.Slice(matches, func(i, j int) bool { return matches[i].score > matches[j].score })
	if len(matches) > 5 {
		matches = matches[:5]
	}
	if len(matches) == 0 {
		return
	}
	fmt.Fprintln(out, "\n## Related notes")
	for _, m := range matches {
		fmt.Fprintf(out, "- [[%s|%s]]\n", m.n.ID, m.n.Title)
	}
}
