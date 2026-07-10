package cmd

import (
	"fmt"
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
	"pi": func() (string, error) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".pi", "agent", "skills"), nil
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
  pi       ~/.pi/agent/skills/

Use --dest to specify a custom destination directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dest == "" {
				if forLLM == "" {
					forLLM = "claude"
				}
				fn, ok := skillsDestinations[forLLM]
				if !ok {
					return fmt.Errorf("install-skills: unknown --for value %q (valid: claude, cursor, zed, pi)", forLLM)
				}
				var err error
				dest, err = fn()
				if err != nil {
					return fmt.Errorf("install-skills: resolve dest: %w", err)
				}
			}

			if listOnly {
				entries, err := nnSkills.FS.ReadDir(".")
				if err != nil {
					return fmt.Errorf("install-skills: read embedded skills: %w", err)
				}
				for _, e := range entries {
					if !e.IsDir() {
						continue
					}
					fmt.Fprintf(outWriter(cmd), "%s\n", e.Name())
				}
				return nil
			}
			if err := installSkillsToDest(dest); err != nil {
				return err
			}
			fmt.Fprintf(outWriter(cmd), "Installed nn stub: %s\n", filepath.Join(dest, "nn", "SKILL.md"))
			return nil
		},
	}

	cmd.Flags().StringVar(&dest, "dest", "", "Custom destination directory (overrides --for)")
	cmd.Flags().StringVar(&forLLM, "for", "", "Target LLM preset: claude (default), cursor, zed, pi")
	cmd.Flags().BoolVar(&listOnly, "list", false, "List skills without copying")
	cmd.Flags().BoolVar(&listOnly, "dry-run", false, "Alias for --list")
	return cmd
}

const nnSkillStub = `---
name: nn
description: nn Zettelkasten CLI — capture, link, review, and maintain notes as an LLM agent.
allowed-tools: Bash(nn:*)
---

# nn

If you have not yet run ` + "`nn skills list`" + ` this session, run it before any ` + "`nn skills get`" + `
or nn command — its output tells you which skill to load and when:

` + "```bash" + `
nn skills list
` + "```" + `

Then load the matching skill before responding:

` + "```bash" + `
nn skills get <name>
` + "```" + `
`

var deprecatedSkillDirs = []string{
	"nn-workflow", "nn-guide", "nn-capture-discipline",
	"nn-link-suggester", "nn-refine", "nn-refine-workflow", "nn-session-debrief",
}

func installSkillsToDest(dest string) error {
	for _, name := range deprecatedSkillDirs {
		path := filepath.Join(dest, name)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			_ = os.RemoveAll(path)
		}
	}
	stubDir := filepath.Join(dest, "nn")
	if err := os.MkdirAll(stubDir, 0o755); err != nil {
		return fmt.Errorf("create stub dir: %w", err)
	}
	return os.WriteFile(filepath.Join(stubDir, "SKILL.md"), []byte(nnSkillStub), 0o644)
}
