package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newLinkCmd(state *rootState) *cobra.Command {
	var annotation string
	var linkType string

	cmd := &cobra.Command{
		Use:   "link <from-id> <to-id>",
		Short: "Add an annotated link between two notes",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if annotation == "" {
				return fmt.Errorf("--annotation is required")
			}
			if linkType == "" {
				return fmt.Errorf("--type is required")
			}
			fromID, toID := args[0], args[1]
			if !note.IsKnownLinkType(linkType) {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: unknown link type %q — known types: refines, contradicts, source-of, extends, supports, questions, governs\n", linkType)
			}
			if err := state.backend.AddLink(fromID, toID, annotation, linkType); err != nil {
				return fmt.Errorf("link: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "linked %s → %s\n", fromID, toID)
			return nil
		},
	}
	cmd.Flags().StringVar(&annotation, "annotation", "", "Link annotation (required)")
	cmd.Flags().StringVar(&linkType, "type", "", "Link relationship type (e.g. refines, contradicts, source-of)")
	return cmd
}

func newUnlinkCmd(state *rootState) *cobra.Command {
	return &cobra.Command{
		Use:   "unlink <from-id> <to-id>",
		Short: "Remove a link between two notes",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			fromID, toID := args[0], args[1]
			if err := state.backend.RemoveLink(fromID, toID); err != nil {
				return fmt.Errorf("unlink: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "unlinked %s → %s\n", fromID, toID)
			return nil
		},
	}
}
