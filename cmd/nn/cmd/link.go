package cmd

import (
	"fmt"
	"strings"

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
				return fmt.Errorf("invalid --type %q: must be one of %s", linkType, strings.Join(note.LinkTypeOrder, ", "))
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
	cmd.AddCommand(newLinkSetTypeCmd(state))
	return cmd
}

func newLinkSetTypeCmd(state *rootState) *cobra.Command {
	var linkType string
	var annotationMatches string
	cmd := &cobra.Command{
		Use:   "set-type <from-id-or-title> <to-id-or-title>",
		Short: "Assign a type to one legacy untyped relationship",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if linkType == "" {
				return fmt.Errorf("--type is required")
			}
			if !note.IsKnownLinkType(linkType) {
				return fmt.Errorf("invalid --type %q: must be one of %s", linkType, strings.Join(note.LinkTypeOrder, ", "))
			}
			fromNote, err := resolveNote(state, args[0])
			if err != nil {
				return fmt.Errorf("link set-type: %w", err)
			}
			toNote, err := resolveNote(state, args[1])
			if err != nil {
				return fmt.Errorf("link set-type: %w", err)
			}
			if err := state.backend.SetLinkType(fromNote.ID, toNote.ID, annotationMatches, linkType); err != nil {
				return fmt.Errorf("link set-type: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "typed link %s → %s as %s\n", fromNote.ID, toNote.ID, linkType)
			return nil
		},
	}
	cmd.Flags().StringVar(&linkType, "type", "", "Known relationship type to assign (required)")
	cmd.Flags().StringVar(&annotationMatches, "annotation-matches", "", "Select an untyped relationship whose annotation contains this text")
	return cmd
}

func newUnlinkCmd(state *rootState) *cobra.Command {
	var linkType string

	cmd := &cobra.Command{
		Use:   "unlink <from-id-or-title> <to-id-or-title>",
		Short: "Remove a link between two notes",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			fromNote, err := resolveNote(state, args[0])
			if err != nil {
				return fmt.Errorf("unlink: %w", err)
			}
			toID := args[1]
			if toNote, err := resolveNote(state, args[1]); err == nil {
				toID = toNote.ID
			}
			if linkType != "" {
				if err := state.backend.RemoveLinkByType(fromNote.ID, toID, linkType); err != nil {
					return fmt.Errorf("unlink: %w", err)
				}
				fmt.Fprintf(outWriter(cmd), "unlinked %s → %s [%s]\n", fromNote.ID, toID, linkType)
			} else {
				if err := state.backend.RemoveLink(fromNote.ID, toID); err != nil {
					return fmt.Errorf("unlink: %w", err)
				}
				fmt.Fprintf(outWriter(cmd), "unlinked %s → %s\n", fromNote.ID, toID)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&linkType, "type", "", "Remove only edges with this link type (e.g. refines, extends)")
	return cmd
}
