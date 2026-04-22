package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newLinkCmd(state *rootState) *cobra.Command {
	var annotation string
	var linkType string
	var linkStatus string

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
			if linkStatus != "draft" && linkStatus != "reviewed" {
				return fmt.Errorf("--status must be draft or reviewed")
			}
			fromNote, err := resolveNote(state, args[0])
			if err != nil {
				return fmt.Errorf("link: %w", err)
			}
			toNote, err := resolveNote(state, args[1])
			if err != nil {
				return fmt.Errorf("link: %w", err)
			}
			if !note.IsKnownLinkType(linkType) {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: unknown link type %q — known types: refines, contradicts, source-of, extends, supports, questions, governs\n", linkType)
			}
			if err := state.backend.AddLink(fromNote.ID, toNote.ID, annotation, linkType, linkStatus); err != nil {
				return fmt.Errorf("link: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "linked %s → %s\n", fromNote.ID, toNote.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&annotation, "annotation", "", "Link annotation (required)")
	cmd.Flags().StringVar(&linkType, "type", "", "Link relationship type (e.g. refines, contradicts, source-of)")
	cmd.Flags().StringVar(&linkStatus, "status", "draft", "Link status: draft or reviewed")
	return cmd
}

func newUnlinkCmd(state *rootState) *cobra.Command {
	return &cobra.Command{
		Use:   "unlink <from-id-or-title> <to-id-or-title>",
		Short: "Remove a link between two notes",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			fromNote, err := resolveNote(state, args[0])
			if err != nil {
				return fmt.Errorf("unlink: %w", err)
			}
			toNote, err := resolveNote(state, args[1])
			if err != nil {
				return fmt.Errorf("unlink: %w", err)
			}
			if err := state.backend.RemoveLink(fromNote.ID, toNote.ID); err != nil {
				return fmt.Errorf("unlink: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "unlinked %s → %s\n", fromNote.ID, toNote.ID)
			return nil
		},
	}
}
