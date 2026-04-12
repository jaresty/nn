package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	nnSkills "github.com/jaresty/nn/skills"
)

// skillsDestinations maps --for preset names to their default skill directories.
// The value is a function so HOME is resolved at call time, not package init.
var skillsDestinations = map[string]func() (string, error){
	"claude": func() (string, error) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".claude", "skills"), nil
	},
	"cursor": func() (string, error) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".cursor", "skills"), nil
	},
	"zed": func() (string, error) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", "zed", "skills"), nil
	},
}

func newInstallSkillsCmd() *cobra.Command {
	var (
		dest     string
		forLLM   string
		listOnly bool
	)

	cmd := &cobra.Command{
		Use:   "install-skills",
		Short: "Copy nn skills into an LLM's skills directory",
		Long: `Copy nn skills into an LLM's skills directory.

Presets (--for):
  claude   ~/.claude/skills/         (default)
  cursor   ~/.cursor/skills/
  zed      ~/.config/zed/skills/

Use --dest to specify a custom destination directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dest == "" {
				if forLLM == "" {
					forLLM = "claude"
				}
				fn, ok := skillsDestinations[forLLM]
				if !ok {
					return fmt.Errorf("install-skills: unknown --for value %q (valid: claude, cursor, zed)", forLLM)
				}
				var err error
				dest, err = fn()
				if err != nil {
					return fmt.Errorf("install-skills: resolve dest: %w", err)
				}
			}

			entries, err := nnSkills.FS.ReadDir(".")
			if err != nil {
				return fmt.Errorf("install-skills: read embedded skills: %w", err)
			}

			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				name := e.Name()
				fmt.Fprintf(outWriter(cmd), "%s\n", name)
				if listOnly {
					continue
				}
				destDir := filepath.Join(dest, name)
				if err := copySkill(name, destDir); err != nil {
					return fmt.Errorf("install-skills: copy %s: %w", name, err)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dest, "dest", "", "Custom destination directory (overrides --for)")
	cmd.Flags().StringVar(&forLLM, "for", "", "Target LLM preset: claude (default), cursor, zed")
	cmd.Flags().BoolVar(&listOnly, "list", false, "List skills without copying")
	cmd.Flags().BoolVar(&listOnly, "dry-run", false, "Alias for --list")
	return cmd
}

// copySkill copies the embedded skill directory to destDir.
func copySkill(name, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	return fs.WalkDir(nnSkills.FS, name, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(name, path)
		dst := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		data, err := nnSkills.FS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	})
}
