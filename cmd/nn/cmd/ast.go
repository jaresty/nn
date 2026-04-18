package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/ast"
)

func newAstCmd(state *rootState) *cobra.Command {
	var jsonOut bool
	var trace bool
	var traceRoot string

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

			if trace {
				root := traceRoot
				if root == "" {
					root = "."
				}
				// Collect unique non-import symbol names.
				seen := make(map[string]bool)
				for _, sym := range f.Symbols {
					if sym.Kind == "import" || sym.Name == "" {
						continue
					}
					if seen[sym.Name] {
						continue
					}
					seen[sym.Name] = true
					fmt.Fprintf(w, "\nreferences to %q (name-match only — not symbol-resolved, may include false positives):\n", sym.Name)
					if err := traceNameMatch(w, root, sym.Name); err != nil {
						return fmt.Errorf("ast --trace: %w", err)
					}
				}
			}
			return nil

		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON array of symbols")
	cmd.Flags().BoolVar(&trace, "trace", false, "For each symbol in the outline, search for name-match references across --root")
	cmd.Flags().StringVar(&traceRoot, "root", ".", "Root directory for --trace reference search")
	return cmd
}

// traceNameMatch searches for occurrences of name across all source files under root.
func traceNameMatch(w io.Writer, root, name string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}
		if info.IsDir() {
			// Skip hidden dirs and vendor.
			base := filepath.Base(path)
			if base == ".git" || base == "vendor" || strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		// Only scan source-like files.
		if !isSourceFile(path) {
			return nil
		}

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
				fmt.Fprintf(w, "  %s:%d\t%s\n", path, lineNum, strings.TrimSpace(line))
			}
		}
		return nil
	})
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
