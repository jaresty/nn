package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newShowCmd(state *rootState) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id-or-title>",
		Short: "Print note content to stdout (accepts ID or title substring)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			// Try exact ID match first.
			n, err := state.backend.Read(query)
			if err != nil {
				// Fall back to case-insensitive title substring match.
				all, listErr := state.backend.List()
				if listErr != nil {
					return fmt.Errorf("show: %w", err)
				}
				var matches []*struct{ id, title string }
				for _, candidate := range all {
					if strings.Contains(strings.ToLower(candidate.Title), strings.ToLower(query)) {
						matches = append(matches, &struct{ id, title string }{candidate.ID, candidate.Title})
					}
				}
				switch len(matches) {
				case 0:
					return fmt.Errorf("show: no note found matching %q", query)
				case 1:
					n, err = state.backend.Read(matches[0].id)
					if err != nil {
						return fmt.Errorf("show: %w", err)
					}
				default:
					fmt.Fprintf(outWriter(cmd), "ambiguous: %d notes match %q:\n", len(matches), query)
					for _, m := range matches {
						fmt.Fprintf(outWriter(cmd), "  %s  %s\n", m.id, m.title)
					}
					return fmt.Errorf("show: ambiguous query %q — use full ID", query)
				}
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
