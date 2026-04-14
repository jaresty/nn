package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/backend"
)

func newBulkLinkCmd(state *rootState) *cobra.Command {
	var toIDs []string
	var annotations []string

	cmd := &cobra.Command{
		Use:   "bulk-link <from-id>",
		Short: "Add multiple annotated links from one note in a single commit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fromID := args[0]
			if len(toIDs) == 0 {
				return fmt.Errorf("bulk-link: at least one --to is required")
			}
			if len(annotations) == 0 {
				return fmt.Errorf("bulk-link: --annotation is required for each --to")
			}
			if len(toIDs) != len(annotations) {
				return fmt.Errorf("bulk-link: %d --to flags but %d --annotation flags; counts must match", len(toIDs), len(annotations))
			}
			targets := make([]backend.LinkTarget, len(toIDs))
			for i, id := range toIDs {
				targets[i] = backend.LinkTarget{ToID: id, Annotation: annotations[i]}
			}
			if err := state.backend.AddLinks(fromID, targets); err != nil {
				return fmt.Errorf("bulk-link: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "linked %s → %d notes\n", fromID, len(targets))
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&toIDs, "to", nil, "Target note ID (repeatable)")
	cmd.Flags().StringArrayVar(&annotations, "annotation", nil, "Link annotation (repeatable, paired with --to)")
	return cmd
}
