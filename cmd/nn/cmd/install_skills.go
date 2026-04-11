package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	nnSkills "github.com/jaresty/nn/skills"
)

func newInstallSkillsCmd() *cobra.Command {
	var (
		dest    string
		listOnly bool
	)

	cmd := &cobra.Command{
		Use:   "install-skills",
		Short: "Copy nn skills into ~/.claude/skills/",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dest == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("install-skills: home dir: %w", err)
				}
				dest = filepath.Join(home, ".claude", "skills")
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

	cmd.Flags().StringVar(&dest, "dest", "", "Destination directory (default: ~/.claude/skills/)")
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
