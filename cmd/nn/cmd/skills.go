package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	nnSkills "github.com/jaresty/nn/skills"
)

func newSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "List or retrieve embedded nn skills",
	}
	cmd.AddCommand(newSkillsListCmd(), newSkillsGetCmd())
	return cmd
}

func newSkillsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available embedded skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := nnSkills.FS.ReadDir(".")
			if err != nil {
				return fmt.Errorf("read embedded skills: %w", err)
			}
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				name := e.Name()
				desc := skillDescription(name)
				fmt.Fprintf(outWriter(cmd), "  %-28s %s\n", name, desc)
			}
			return nil
		},
	}
}

func newSkillsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <name>",
		Short: "Print full content of a named skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			data, err := nnSkills.FS.ReadFile(name + "/SKILL.md")
			if err != nil {
				return fmt.Errorf("skill %q not found", name)
			}
			_, err = fmt.Fprint(outWriter(cmd), string(data))
			return err
		},
	}
}

// skillDescription extracts the description field from a skill's SKILL.md frontmatter.
func skillDescription(name string) string {
	data, err := nnSkills.FS.ReadFile(name + "/SKILL.md")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "description:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}
	return ""
}
