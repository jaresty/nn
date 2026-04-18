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
	var linkStatus string

	cmd := &cobra.Command{
		Use:   "bulk-update-link <from-id>",
		Short: "Update type, annotation, and/or status of multiple existing links in a single commit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fromID := args[0]
			if len(toIDs) == 0 {
				return fmt.Errorf("bulk-update-link: at least one --to is required")
			}
			hasTypes := len(types) > 0
			hasAnnotations := len(annotations) > 0
			hasStatus := cmd.Flags().Changed("status")
			if hasTypes && len(types) != len(toIDs) {
				return fmt.Errorf("bulk-update-link: %d --to flags but %d --type flags; counts must match", len(toIDs), len(types))
			}
			if hasAnnotations && len(annotations) != len(toIDs) {
				return fmt.Errorf("bulk-update-link: %d --to flags but %d --annotation flags; counts must match", len(toIDs), len(annotations))
			}
			if !hasTypes && !hasAnnotations && !hasStatus {
				return fmt.Errorf("bulk-update-link: at least one of --type, --annotation, or --status is required")
			}
			if hasStatus && linkStatus != "draft" && linkStatus != "reviewed" {
				return fmt.Errorf("bulk-update-link: --status must be draft or reviewed")
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
				if hasStatus {
					s := linkStatus
					updates[i].Status = &s
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
	cmd.Flags().StringVar(&linkStatus, "status", "", "Link status to apply to all --to links: draft or reviewed")
	return cmd
}
