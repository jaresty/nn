package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newShowCmd(state *rootState) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Print note content to stdout",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := state.backend.Read(args[0])
			if err != nil {
				return fmt.Errorf("show: %w", err)
			}
			data, err := n.Marshal()
			if err != nil {
				return fmt.Errorf("show: marshal: %w", err)
			}
			fmt.Fprint(outWriter(cmd), string(data))
			return nil
		},
	}
}
