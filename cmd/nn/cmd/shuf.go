package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

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
		Long:  "Sample random units from files (or stdin) and show BM25-matched notes. Binary files are skipped and summarized on stderr. Sample units larger than 65536 bytes are also skipped and summarized. Piped stdin is not MIME-filtered, but its sampled units use the same size limit.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShufWithDiagnostics(cmd.InOrStdin(), args, cmd.OutOrStdout(), cmd.ErrOrStderr(), state, count, unit)
		},
	}
	cmd.Flags().IntVar(&count, "count", 5, "Number of units to sample")
	cmd.Flags().StringVar(&unit, "unit", "paragraphs", "Sampling unit: lines, paragraphs, or symbols")
	return cmd
}

func runShuf(stdin io.Reader, paths []string, out io.Writer, state *rootState, count int, unit string) error {
	return runShufWithDiagnostics(stdin, paths, out, io.Discard, state, count, unit)
}

const maxShufUnitBytes = 64 << 10

func runShufWithDiagnostics(stdin io.Reader, paths []string, out, errOut io.Writer, state *rootState, count int, unit string) error {
	units, binarySkips, err := collectUnits(stdin, paths, unit)
	if err != nil {
		return err
	}
	if binarySkips > 0 {
		noun := "files"
		if binarySkips == 1 {
			noun = "file"
		}
		fmt.Fprintf(errOut, "nn shuf: skipped %d binary %s\n", binarySkips, noun)
	}
	safeUnits := units[:0]
	oversizedSkips := 0
	for _, candidate := range units {
		if len([]byte(candidate)) > maxShufUnitBytes {
			oversizedSkips++
			continue
		}
		safeUnits = append(safeUnits, candidate)
	}
	units = safeUnits
	if oversizedSkips > 0 {
		noun := "units"
		if oversizedSkips == 1 {
			noun = "unit"
		}
		fmt.Fprintf(errOut, "nn shuf: skipped %d oversized %s (>65536 bytes)\n", oversizedSkips, noun)
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
	if state != nil && state.backend != nil {
		notes, _ = state.backend.List()
	}
	// Prepare the query-invariant corpus inputs once; each sample only varies
	// the query, so the link maps and fieldIDF must not be recomputed per sample.
	var prepared preparedCorpus
	if len(notes) > 0 {
		prepared = prepareCorpus(notes, state.notebookDir)
	}

	for _, idx := range indices {
		u := units[idx]
		fmt.Fprintln(out, "---")
		fmt.Fprintln(out, u)

		if len(notes) > 0 && strings.TrimSpace(u) != "" {
			printShufRelated(out, prepared, notes, u)
		}
	}
	return nil
}

func collectUnits(stdin io.Reader, paths []string, unit string) ([]string, int, error) {
	if len(paths) == 0 {
		if stdin == nil {
			return nil, 0, nil
		}
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, 0, fmt.Errorf("shuf: read stdin: %w", err)
		}
		return splitUnits(string(data), unit, ""), 0, nil
	}

	var all []string
	binarySkips := 0
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, 0, fmt.Errorf("shuf: stat %s: %w", p, err)
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
				if !isShufTextContent(data) {
					binarySkips++
					return nil
				}
				all = append(all, splitUnits(string(data), unit, path)...)
				return nil
			}); err != nil {
				return nil, 0, err
			}
		} else {
			data, err := os.ReadFile(p)
			if err != nil {
				return nil, 0, fmt.Errorf("shuf: read %s: %w", p, err)
			}
			if !isShufTextContent(data) {
				binarySkips++
				continue
			}
			all = append(all, splitUnits(string(data), unit, p)...)
		}
	}
	return all, binarySkips, nil
}

func isShufTextContent(data []byte) bool {
	trimmed := bytes.TrimLeft(data, " \t\n\f\r")
	return utf8.Valid(data) && !hasForbiddenShufControl(data) && isTextContent(trimmed)
}

func hasForbiddenShufControl(data []byte) bool {
	for _, b := range data {
		if b == 0x7f || (b < 0x20 && b != '\t' && b != '\n' && b != '\f' && b != '\r') {
			return true
		}
	}
	return false
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
	scanner.Buffer(make([]byte, 64<<10), len(content)+1)
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
	scanner.Buffer(make([]byte, 64<<10), len(content)+1)
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

func printShufRelated(out io.Writer, prepared preparedCorpus, notes []*note.Note, query string) {
	scores := prepared.rankedByQuery(notes, query)
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
