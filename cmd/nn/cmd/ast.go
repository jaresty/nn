package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/ast"
	"github.com/jaresty/nn/internal/note"
)

func newAstCmd(state *rootState) *cobra.Command {
	var jsonOut bool
	var refs bool
	var refsRoot string
	var refsSymbols []string

	cmd := &cobra.Command{
		Use:   "ast <file>",
		Short: "Print a compact structural outline of a source file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			// Resolve relative to notebook dir if not absolute.
			if !filepath.IsAbs(filePath) {
				// Try relative to cwd first, then relative to notebook.
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					filePath = filepath.Join(state.notebookDir, filePath)
				}
			}

			f, err := ast.Parse(filePath)
			if err != nil {
				return fmt.Errorf("ast: %w", err)
			}

			w := outWriter(cmd)

			if jsonOut {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(f.Symbols)
			}

			fmt.Fprintf(w, "file: %s  language: %s\n", args[0], f.Language)
			for _, sym := range f.Symbols {
				if sym.Kind == "import" {
					fmt.Fprintf(w, "imports: %s\n", sym.Name)
					continue
				}
				fmt.Fprintf(w, "%s\n", sym.Signature)
			}

			// BM25 annotation: query each non-import symbol name against nn notes.
			sessionReads := loadSessionReads(resolveCfgDir())
			notes, _ := state.backend.List()
			if len(notes) > 0 {
				allInbound := make(map[string][]string)
				for _, n := range notes {
					for _, lnk := range n.Links {
						allInbound[lnk.TargetID] = append(allInbound[lnk.TargetID], lnk.Annotation)
					}
				}
				seenNotes := map[string]bool{}
				var relatedNotes []*note.Note
				for _, sym := range f.Symbols {
					if sym.Kind == "import" || sym.Name == "" {
						continue
					}
					query := sym.Body
					if query == "" {
						query = sym.Name
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
					sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
					if len(ranked) > 2 {
						ranked = ranked[:2]
					}
					for _, r := range ranked {
						if !seenNotes[r.n.ID] {
							seenNotes[r.n.ID] = true
							relatedNotes = append(relatedNotes, r.n)
						}
					}
				}
				fmt.Fprintln(w, "\n## Related notes")
				hasUnread := false
				for _, n := range relatedNotes {
					readMarker := ""
					if sessionReads[n.ID] {
						readMarker = " [read]"
					} else {
						hasUnread = true
					}
					fmt.Fprintf(w, "- [[%s|%s]] [likely relevant]%s\n", n.ID, n.Title, readMarker)
				}
				printResolveInstruction(w, hasUnread)
			} else {
				fmt.Fprintln(w, "\n## Related notes")
			}

			if refs {
				root := refsRoot
				if root == "" {
					// Default to the directory of the file being analyzed so that
					// `nn ast /abs/path/file.go --refs` searches the right tree
					// regardless of process cwd.
					root = filepath.Dir(filePath)
				}
				// Normalize to absolute path so filepath.Base("..") doesn't trigger
				// the hidden-dir skip guard inside collectNameMatches.
				if abs, err := filepath.Abs(root); err == nil {
					root = abs
				}
				// Build filter set from --symbol flags.
				symbolFilter := make(map[string]bool, len(refsSymbols))
				for _, s := range refsSymbols {
					symbolFilter[s] = true
				}

				// Collect unique non-import symbol names to query.
				seen := make(map[string]bool)
				var targets []string
				for _, sym := range f.Symbols {
					if sym.Kind == "import" || sym.Name == "" {
						continue
					}
					if seen[sym.Name] {
						continue
					}
					if len(symbolFilter) > 0 && !symbolFilter[sym.Name] {
						continue
					}
					seen[sym.Name] = true
					targets = append(targets, sym.Name)
				}

				totalMatches := 0
				withRefs := 0
				filesScanned := 0
				for i, name := range targets {
					var matches []string
					var scanned int
					matches, scanned = collectNameMatches(root, name)
					if i == 0 {
						filesScanned = scanned
					}
					if len(matches) == 0 {
						continue
					}
					withRefs++
					totalMatches += len(matches)
					fmt.Fprintf(w, "\nreferences to %q — %d match(es) (name-match only, may include false positives):\n", name, len(matches))
					for _, m := range matches {
						fmt.Fprintln(w, m)
					}
				}
				fmt.Fprintf(w, "\nsummary: %d symbol(s) · %d with references · %d match(es) · %d files scanned\n",
					len(targets), withRefs, totalMatches, filesScanned)
			}
			return nil

		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON array of symbols")
	cmd.Flags().BoolVar(&refs, "refs", false, "For each symbol in the outline, search for name-match references across --root")
	cmd.Flags().StringVar(&refsRoot, "root", "", "Root directory for --refs reference search (default: directory of the analyzed file)")
	cmd.Flags().StringArrayVar(&refsSymbols, "symbol", nil, "Limit --refs to these symbol names (repeatable); default: all symbols")
	return cmd
}

// collectNameMatches returns all "path:line  content" strings where name appears,
// plus the number of files scanned.
func collectNameMatches(root, name string) (matches []string, filesScanned int) {
	walkErr := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "vendor" || (len(base) > 1 && base[0] == '.' && base[1] != '.') {
				return filepath.SkipDir
			}
			return nil
		}
		if !isSourceFile(path) {
			return nil
		}
		filesScanned++

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if strings.Contains(line, name) {
				matches = append(matches, fmt.Sprintf("  %s:%d\t%s", path, lineNum, strings.TrimSpace(line)))
			}
		}
		return nil
	})
	_ = walkErr
	return
}

func isSourceFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go", ".py", ".js", ".ts", ".rs", ".java", ".rb", ".c", ".cpp", ".cc",
		".cs", ".swift", ".kt", ".scala", ".lua", ".sh", ".bash", ".r",
		".yaml", ".yml", ".toml", ".json", ".md":
		return true
	}
	return false
}
