package cmd

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/jaresty/nn/internal/note"
	"github.com/jaresty/nn/internal/trace"
	"github.com/odvcencio/gotreesitter/grammars"
	"github.com/spf13/cobra"
)

func newGrepCmd(state *rootState) *cobra.Command {
	var contextLines int
	var notesPerMatch int
	var traceFlag bool
	var maxMatches int

	cmd := &cobra.Command{
		Use:   "grep <pattern> [path...]",
		Short: "Search files for pattern and annotate matches with related nn notes",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pattern := args[0]
			re, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("invalid pattern %q: %w", pattern, err)
			}
			searchPaths := args[1:]
			if len(searchPaths) == 0 {
				searchPaths = []string{"."}
			}

			// Collect lines from file(s).
			type fileLine struct {
				file    string
				lineNum int
				text    string
			}
			var allLines []fileLine

			var files []string
			for _, searchPath := range searchPaths {
				info, err := os.Stat(searchPath)
				if err != nil {
					return fmt.Errorf("stat %s: %w", searchPath, err)
				}
				if info.IsDir() {
					if err := gitTrackedFiles(searchPath, &files); err != nil {
						if err := collectFiles(searchPath, &files); err != nil {
							return err
						}
					}
				} else {
					files = append(files, searchPath)
				}
			}

			// fileStartIdx maps file path → first index in allLines for that file.
			fileStartIdx := make(map[string]int)
			fileEndIdx := make(map[string]int)
			const maxFileSize = 1 << 20 // 1 MB
			for _, f := range files {
				if fi, err := os.Stat(f); err == nil && fi.Size() > maxFileSize {
					fmt.Fprintf(cmd.ErrOrStderr(), "nn grep: skipping large file %s (%d bytes)\n", f, fi.Size())
					continue
				}
				lines, err := readFileLines(f)
				if err != nil {
					continue
				}
				fileStartIdx[f] = len(allLines)
				for i, line := range lines {
					allLines = append(allLines, fileLine{f, i + 1, line})
				}
				fileEndIdx[f] = len(allLines)
			}

			// Find matches.
			type match struct {
				file         string
				lineNum      int
				text         string
				context      []string
				contextStart int // 1-based line number of context[0]
			}
			var matches []match
			for idx, fl := range allLines {
				if !re.MatchString(fl.text) {
					continue
				}
				// Gather context lines, clamped to the current file's boundary.
				fStart := fileStartIdx[fl.file]
				fEnd := fileEndIdx[fl.file]
				start := idx - contextLines
				if start < fStart {
					start = fStart
				}
				end := idx + contextLines + 1
				if end > fEnd {
					end = fEnd
				}
				var ctx []string
				for _, cl := range allLines[start:end] {
					ctx = append(ctx, cl.text)
				}
				matches = append(matches, match{fl.file, fl.lineNum, fl.text, ctx, allLines[start].lineNum})
			}

			if len(matches) == 0 {
				return nil
			}

			truncated := 0
			if maxMatches > 0 && len(matches) > maxMatches {
				truncated = len(matches) - maxMatches
				matches = matches[:maxMatches]
			}

			// Load session reads once for this invocation.
			sessionReads := loadSessionReads(resolveCfgDir())
			hasUnread := false

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

			traceSuggested := map[string]bool{}
			prevEnd := -1 // last line number printed in previous group (for separator logic)
			currentFile := ""
			for _, m := range matches {
				// Print file header when file changes.
				if m.file != currentFile {
					if currentFile != "" {
						fmt.Fprintln(w)
					}
					fmt.Fprintf(w, "==> %s <==\n", m.file)
					currentFile = m.file
					prevEnd = -1
				}
				// Print separator when context windows don't overlap with previous group.
				if contextLines > 0 && prevEnd >= 0 && m.contextStart > prevEnd+1 {
					fmt.Fprintln(w, "--")
				}
				// Print before-context lines.
				for i, line := range m.context {
					lineNum := m.contextStart + i
					if lineNum == m.lineNum {
						break
					}
					fmt.Fprintf(w, "%d:%s\n", lineNum, line)
				}
				// Print the match line.
				fmt.Fprintf(w, "%d:%s\n", m.lineNum, m.text)
				// Print after-context lines.
				afterStart := m.lineNum - m.contextStart + 1
				for i := afterStart; i < len(m.context); i++ {
					lineNum := m.contextStart + i
					fmt.Fprintf(w, "%d:%s\n", lineNum, m.context[i])
				}
				prevEnd = m.contextStart + len(m.context) - 1

				if !traceSuggested[m.file] && isTraceableFile(m.file) && traceFlag {
					traceSuggested[m.file] = true
					if traceFlag {
						dir := filepath.Dir(m.file)
						if idx, err := trace.BuildIndex(dir); err == nil {
							if sym, _, ok := resolveFileLineInIndex(idx, m.file, m.lineNum); ok {
								result := trace.Trace(idx, []string{sym}, 3, notes)
								fmt.Fprintf(w, "  [trace: %s --symbol %s]\n", dir, sym)
								for _, n := range result.Nodes {
									fmt.Fprintf(w, "    %s (%s) [%s:%d]\n", n.Name, n.Kind, n.File, n.Line)
								}
							}
						}
					} else {
						fmt.Fprintf(w, "  → To follow the execution path: nn trace %s --symbol <name>\n", filepath.Dir(m.file))
						fmt.Fprintf(w, "    (trace follows callable syntax but may not resolve TypeScript types, re-exported symbols, or fluent framework composition)\n")
					}
				}

				query := strings.Join(m.context, " ")
				if strings.TrimSpace(query) == "" {
					continue
				}
				scores := RankedByQuery(notes, allInbound, query, state.notebookDir)

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
					readMarker := ""
					if sessionReads[r.n.ID] {
						readMarker = " [read]"
					} else {
						hasUnread = true
					}
					fmt.Fprintf(w, "  → [[%s|%s]] %s%s\n", r.n.ID, r.n.Title, scoreLabel(r.score), readMarker)
				}
			}
			if truncated > 0 {
				fmt.Fprintf(w, "truncated: %d more matches not shown (use --max-matches to adjust)\n", truncated)
			}
			printResolveInstruction(w, hasUnread)
			return nil
		},
	}

	cmd.Flags().IntVar(&contextLines, "context", 3, "Number of surrounding lines to include in BM25 query")
	cmd.Flags().IntVar(&notesPerMatch, "notes-per-match", 2, "Maximum related notes to show per match")
	cmd.Flags().IntVar(&maxMatches, "max-matches", 50, "Maximum number of matches to show (0 = unlimited)")
	cmd.Flags().BoolVar(&traceFlag, "trace", false, "Invoke nn trace inline for each traceable matched file")
	return cmd
}

func gitTrackedFiles(dir string, out *[]string) error {
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard")
	cmd.Dir = dir
	b, err := cmd.Output()
	if err != nil {
		return err
	}
	for _, line := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		if line == "" {
			continue
		}
		*out = append(*out, filepath.Join(dir, line))
	}
	return nil
}

func collectFiles(dir string, out *[]string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() == ".git" {
			continue
		}
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

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return nil, err
	}
	if !strings.HasPrefix(http.DetectContentType(buf[:n]), "text/") {
		return nil, nil
	}
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}

func isTraceableFile(path string) bool {
	return grammars.DetectLanguage(path) != nil
}

func resolveFileLineInIndex(idx *trace.Index, file string, lineNum int) (string, string, bool) {
	for _, sites := range idx.ByName {
		for _, s := range sites {
			if s.File == file && s.StartLine <= lineNum && s.EndLine >= lineNum {
				return s.Name, filepath.Dir(file), true
			}
		}
	}
	return "", "", false
}
