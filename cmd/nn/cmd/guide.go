package cmd

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/spf13/cobra"

	nnSkills "github.com/jaresty/nn/skills"
)

func newGuideCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "guide [topic]",
		Short: "Show workflow guidance (type: guide, workflow)",
		Long: `Show embedded skill documentation for nn.

Available topics:
  guide     — type selection, command reference, linking conventions
  workflow  — full agentic workflow with session-start protocol loading

Examples:
  nn guide            # list available topics
  nn guide workflow   # print the full workflow guide`,
		Args: cobra.MaximumNArgs(1),
		// Does not require a configured notebook.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			w := outWriter(cmd)

			if len(args) == 0 {
				// List available topics from embedded skills.
				entries, err := nnSkills.FS.ReadDir(".")
				if err != nil {
					return fmt.Errorf("guide: %w", err)
				}
				fmt.Fprintln(w, "Available topics (nn guide <topic>):")
				for _, e := range entries {
					if e.IsDir() {
						topic := strings.TrimPrefix(e.Name(), "nn-")
						fmt.Fprintf(w, "  %-12s  %s\n", topic, skillOneLiner(e.Name()))
					}
				}
				return nil
			}

			topic := args[0]
			skillName := "nn-" + topic
			skillPath := skillName + "/SKILL.md"
			data, err := fs.ReadFile(nnSkills.FS, skillPath)
			if err != nil {
				return fmt.Errorf("guide: unknown topic %q — run 'nn guide' to list available topics", topic)
			}
			fmt.Fprint(w, string(data))
			return nil
		},
	}
}

// skillOneLiner returns a brief description for known skill names.
func skillOneLiner(skillName string) string {
	switch skillName {
	case "nn-guide":
		return "type selection, command reference, linking conventions"
	case "nn-workflow":
		return "full agentic workflow with session-start protocol loading"
	default:
		return ""
	}
}
