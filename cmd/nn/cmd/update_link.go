package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUpdateLinkCmd(state *rootState) *cobra.Command {
	var annotation string
	var linkType string

	cmd := &cobra.Command{
		Use:   "update-link <from-id> <to-id>",
		Short: "Update annotation or type of an existing link",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			annChanged := cmd.Flags().Changed("annotation")
			typeChanged := cmd.Flags().Changed("type")
			if !annChanged && !typeChanged {
				return fmt.Errorf("at least one of --annotation or --type is required")
			}
			fromID, toID := args[0], args[1]
			var annPtr, typePtr *string
			if annChanged {
				annPtr = &annotation
			}
			if typeChanged {
				typePtr = &linkType
			}
			if err := state.backend.UpdateLink(fromID, toID, annPtr, typePtr); err != nil {
				return fmt.Errorf("update-link: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "updated link %s → %s\n", fromID, toID)
			return nil
		},
	}
	cmd.Flags().StringVar(&annotation, "annotation", "", "New link annotation")
	cmd.Flags().StringVar(&linkType, "type", "", "New link type (e.g. refines, contradicts)")
	return cmd
}
