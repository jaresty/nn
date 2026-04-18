package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUpdateLinkCmd(state *rootState) *cobra.Command {
	var annotation string
	var linkType string
	var linkStatus string

	cmd := &cobra.Command{
		Use:   "update-link <from-id> <to-id>",
		Short: "Update annotation, type, or status of an existing link",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			annChanged := cmd.Flags().Changed("annotation")
			typeChanged := cmd.Flags().Changed("type")
			statusChanged := cmd.Flags().Changed("status")
			if !annChanged && !typeChanged && !statusChanged {
				return fmt.Errorf("at least one of --annotation, --type, or --status is required")
			}
			fromID, toID := args[0], args[1]
			var annPtr, typePtr, statusPtr *string
			if annChanged {
				annPtr = &annotation
			}
			if typeChanged {
				typePtr = &linkType
			}
			if statusChanged {
				if linkStatus != "draft" && linkStatus != "reviewed" {
					return fmt.Errorf("--status must be draft or reviewed")
				}
				statusPtr = &linkStatus
			}
			if err := state.backend.UpdateLink(fromID, toID, annPtr, typePtr, statusPtr); err != nil {
				return fmt.Errorf("update-link: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "updated link %s → %s\n", fromID, toID)
			return nil
		},
	}
	cmd.Flags().StringVar(&annotation, "annotation", "", "New link annotation")
	cmd.Flags().StringVar(&linkType, "type", "", "New link type (e.g. refines, contradicts)")
	cmd.Flags().StringVar(&linkStatus, "status", "", "New link status: draft or reviewed")
	return cmd
}
