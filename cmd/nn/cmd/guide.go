package cmd

import (
	"fmt"
	"io/fs"

	"github.com/spf13/cobra"

	nnSkills "github.com/jaresty/nn/skills"
)

// guideTopics maps user-facing topic names to embedded skill directory names.
var guideTopics = []struct {
	topic    string
	skill    string
	oneliner string
}{
	{"ref", "nn-guide", "type selection, command reference, linking conventions"},
	{"workflow", "nn-workflow", "full agentic workflow with session-start protocol loading"},
}

func newGuideCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "guide [topic]",
		Short: "Show workflow guidance (topics: ref, workflow)",
		Long: `Show embedded skill documentation for nn.

Available topics:
  ref       — type selection, command reference, linking conventions
  workflow  — full agentic workflow with session-start protocol loading

Examples:
  nn guide            # list available topics
  nn guide ref        # print the command reference and type guide
  nn guide workflow   # print the full agentic workflow`,
		Args: cobra.MaximumNArgs(1),
		// Does not require a configured notebook.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			w := outWriter(cmd)

			if len(args) == 0 {
				fmt.Fprintln(w, "Available topics (nn guide <topic>):")
				for _, t := range guideTopics {
					fmt.Fprintf(w, "  %-12s  %s\n", t.topic, t.oneliner)
				}
				return nil
			}

			topic := args[0]
			for _, t := range guideTopics {
				if t.topic == topic {
					data, err := fs.ReadFile(nnSkills.FS, t.skill+"/SKILL.md")
					if err != nil {
						return fmt.Errorf("guide: could not read %q: %w", t.skill, err)
					}
					fmt.Fprint(w, string(data))
					return nil
				}
			}
			return fmt.Errorf("guide: unknown topic %q — run 'nn guide' to list available topics", topic)
		},
	}
}
