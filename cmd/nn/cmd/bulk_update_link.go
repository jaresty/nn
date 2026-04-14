package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/backend"
)

func newBulkUpdateLinkCmd(state *rootState) *cobra.Command {
	var toIDs []string
	var types []string
	var annotations []string

	cmd := &cobra.Command{
		Use:   "bulk-update-link <from-id>",
		Short: "Update type and/or annotation of multiple existing links in a single commit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fromID := args[0]
			if len(toIDs) == 0 {
				return fmt.Errorf("bulk-update-link: at least one --to is required")
			}
			hasTypes := len(types) > 0
			hasAnnotations := len(annotations) > 0
			if hasTypes && len(types) != len(toIDs) {
				return fmt.Errorf("bulk-update-link: %d --to flags but %d --type flags; counts must match", len(toIDs), len(types))
			}
			if hasAnnotations && len(annotations) != len(toIDs) {
				return fmt.Errorf("bulk-update-link: %d --to flags but %d --annotation flags; counts must match", len(toIDs), len(annotations))
			}
			if !hasTypes && !hasAnnotations {
				return fmt.Errorf("bulk-update-link: at least one of --type or --annotation is required")
			}

			updates := make([]backend.LinkUpdate, len(toIDs))
			for i, id := range toIDs {
				updates[i] = backend.LinkUpdate{ToID: id}
				if hasTypes {
					t := types[i]
					updates[i].Type = &t
				}
				if hasAnnotations {
					a := annotations[i]
					updates[i].Annotation = &a
				}
			}

			if err := state.backend.BulkUpdateLinks(fromID, updates); err != nil {
				return fmt.Errorf("bulk-update-link: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "updated %d links from %s\n", len(updates), fromID)
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&toIDs, "to", nil, "Target note ID (repeatable)")
	cmd.Flags().StringArrayVar(&types, "type", nil, "Link type (repeatable, paired with --to)")
	cmd.Flags().StringArrayVar(&annotations, "annotation", nil, "Link annotation (repeatable, paired with --to)")
	return cmd
}
