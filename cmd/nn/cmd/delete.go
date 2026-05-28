package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newDeleteCmd(state *rootState) *cobra.Command {
	var confirm bool
	var fromStdin bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a note (warns if linked-to by others); --from-stdin reads IDs line-by-line",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("--confirm required to delete a note")
			}

			if fromStdin {
				scanner := bufio.NewScanner(cmd.InOrStdin())
				for scanner.Scan() {
					id := strings.TrimSpace(scanner.Text())
					if id == "" {
						continue
					}
					if err := deleteOne(cmd, state, id); err != nil {
						return err
					}
				}
				return scanner.Err()
			}

			if len(args) != 1 {
				return fmt.Errorf("delete: provide exactly one ID or use --from-stdin")
			}
			return deleteOne(cmd, state, args[0])
		},
	}
	cmd.Flags().BoolVar(&confirm, "confirm", false, "Confirm deletion")
	cmd.Flags().BoolVar(&fromStdin, "from-stdin", false, "Read note IDs line-by-line from stdin and delete each")
	return cmd
}

func deleteOne(cmd *cobra.Command, state *rootState, query string) error {
	n, err := resolveNote(state, query)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	id := n.ID

	notes, listErr := state.backend.List()
	if listErr != nil {
		return fmt.Errorf("delete: list: %w", listErr)
	}
	var linkers []string
	for _, candidate := range notes {
		if candidate.ID == id {
			continue
		}
		for _, lnk := range candidate.Links {
			if lnk.TargetID == id {
				linkers = append(linkers, candidate.ID)
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
}
