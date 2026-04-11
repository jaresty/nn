package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDeleteCmd(state *rootState) *cobra.Command {
	var confirm bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a note (warns if linked-to by others)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("--confirm required to delete a note")
			}
			id := args[0]

			// Check for inbound links and warn.
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("delete: list: %w", err)
			}
			var linkers []string
			for _, n := range notes {
				if n.ID == id {
					continue
				}
				for _, lnk := range n.Links {
					if lnk.TargetID == id {
						linkers = append(linkers, n.ID)
					}
				}
			}
			if len(linkers) > 0 {
				fmt.Fprintf(outWriter(cmd), "warning: %d note(s) link to %s: %v\n",
					len(linkers), id, linkers)
			}

			if err := state.backend.Delete(id); err != nil {
				return fmt.Errorf("delete: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "deleted %s\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVar(&confirm, "confirm", false, "Confirm deletion")
	return cmd
}
