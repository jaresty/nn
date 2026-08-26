package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/backend"
	"github.com/jaresty/nn/internal/note"
)

func newBulkLinkCmd(state *rootState) *cobra.Command {
	var toIDs []string
	var annotations []string
	var types []string
	var linkStatus string

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
			if len(types) == 0 {
				return fmt.Errorf("bulk-link: --type is required for each --to")
			}
			if len(types) != 1 && len(types) != len(toIDs) {
				return fmt.Errorf("bulk-link: %d --to flags but %d --type flags; counts must match", len(toIDs), len(types))
			}
			if linkStatus != "draft" && linkStatus != "reviewed" {
				return fmt.Errorf("bulk-link: --status must be draft or reviewed")
			}
			for _, linkType := range types {
				if !note.IsKnownLinkType(linkType) {
					return fmt.Errorf("bulk-link: invalid --type %q: must be one of %s", linkType, strings.Join(note.LinkTypeOrder, ", "))
				}
			}
			targets := make([]backend.LinkTarget, len(toIDs))
			for i, id := range toIDs {
				linkType := types[i%len(types)]
				targets[i] = backend.LinkTarget{ToID: id, Annotation: annotations[i], Status: linkStatus, Type: linkType}
			}
			if err := state.backend.AddLinks(fromID, targets); err != nil {
				return fmt.Errorf("bulk-link: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "linked %s -> %d notes\n", fromID, len(targets))
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&toIDs, "to", nil, "Target note ID (repeatable)")
	cmd.Flags().StringArrayVar(&annotations, "annotation", nil, "Link annotation (repeatable, paired with --to)")
	cmd.Flags().StringArrayVar(&types, "type", nil, "Link type: one value broadcasts to all --to entries, or one per --to")
	cmd.Flags().StringVar(&linkStatus, "status", "draft", "Link status for all links: draft or reviewed")
	return cmd
}
