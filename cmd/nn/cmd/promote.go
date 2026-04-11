package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newPromoteCmd(state *rootState) *cobra.Command {
	var to string

	cmd := &cobra.Command{
		Use:   "promote <id>",
		Short: "Advance note status: draft → reviewed → permanent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if to == "" {
				return fmt.Errorf("--to is required (reviewed|permanent)")
			}
			status := note.Status(to)
			if !status.IsValid() {
				return fmt.Errorf("invalid --to %q: must be reviewed|permanent", to)
			}
			id := args[0]
			if err := state.backend.Promote(id, status); err != nil {
				return fmt.Errorf("promote: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "promoted %s to %s\n", id, to)
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "Target status: reviewed|permanent")
	return cmd
}
