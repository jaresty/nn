package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/backend"
)

func newBulkUnlinkCmd(state *rootState) *cobra.Command {
	var toIDs []string
	var types []string

	cmd := &cobra.Command{
		Use:   "bulk-unlink <from-id>",
		Short: "Remove multiple links from one note in a single commit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(toIDs) == 0 {
				return fmt.Errorf("bulk-unlink: at least one --to is required")
			}
			if len(types) != 0 && len(types) != 1 && len(types) != len(toIDs) {
				return fmt.Errorf("bulk-unlink: %d --to flags but %d --type flags; counts must match", len(toIDs), len(types))
			}
			removals := make([]backend.LinkRemoval, len(toIDs))
			for i, toID := range toIDs {
				linkType := ""
				if len(types) > 0 {
					linkType = types[i%len(types)]
				}
				removals[i] = backend.LinkRemoval{ToID: toID, Type: linkType}
			}
			if err := state.backend.RemoveLinks(args[0], removals); err != nil {
				return fmt.Errorf("bulk-unlink: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "unlinked %s → %d notes\n", args[0], len(removals))
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&toIDs, "to", nil, "Target note ID (repeatable)")
	cmd.Flags().StringArrayVar(&types, "type", nil, "Link type: omit to remove all types, one value broadcasts, or one per --to")
	return cmd
}
