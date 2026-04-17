package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newUpdateCmd(state *rootState) *cobra.Command {
	var (
		title   string
		tags    string
		content string
		appendS string
		noEdit  bool
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an existing note's title, tags, or body",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if content != "" && appendS != "" {
				return fmt.Errorf("--content and --append are mutually exclusive")
			}
			if title == "" && tags == "" && content == "" && appendS == "" {
				return fmt.Errorf("at least one of --title, --tags, --content, --append is required")
			}

			id := args[0]
			n, err := state.backend.Read(id)
			if err != nil {
				return fmt.Errorf("update: %w", err)
			}

			if title != "" {
				n.Title = title
			}
			if tags != "" {
				var parsed []string
				for _, t := range strings.Split(tags, ",") {
					if t = strings.TrimSpace(t); t != "" {
						parsed = append(parsed, t)
					}
				}
				n.Tags = parsed
			}
			if content != "" {
				n.Body = content
			}
			if appendS != "" {
				if n.Body == "" {
					n.Body = appendS
				} else {
					n.Body = n.Body + "\n\n" + appendS
				}
			}
			n.Modified = time.Now().UTC()
			warnIfLarge(cmd, n.Body)

			if err := state.backend.Update(n); err != nil {
				return fmt.Errorf("update: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "updated %s\n", n.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "New note title")
	cmd.Flags().StringVar(&tags, "tags", "", "Replace tags (comma-separated)")
	cmd.Flags().StringVar(&content, "content", "", "Replace note body entirely")
	cmd.Flags().StringVar(&appendS, "append", "", "Append text to note body")
	cmd.Flags().BoolVar(&noEdit, "no-edit", false, "Skip opening $EDITOR")
	return cmd
}
