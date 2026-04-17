package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newShowCmd(state *rootState) *cobra.Command {
	var linkedFrom string

	cmd := &cobra.Command{
		Use:   "show <id-or-title> [<id-or-title>...]",
		Short: "Print note content to stdout (accepts ID or title substring)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := outWriter(cmd)

			if linkedFrom != "" {
				src, err := resolveNote(state, linkedFrom)
				if err != nil {
					return fmt.Errorf("show --linked-from: %w", err)
				}
				for i, lnk := range src.Links {
					n, err := state.backend.Read(lnk.TargetID)
					if err != nil {
						continue // skip broken links silently
					}
					if i > 0 {
						fmt.Fprintln(w, "---")
					}
					data, err := n.Marshal()
					if err != nil {
						return fmt.Errorf("show: marshal: %w", err)
					}
					fmt.Fprint(w, string(data))
				}
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("show: provide at least one ID or use --linked-from")
			}

			for i, query := range args {
				if i > 0 {
					fmt.Fprintln(w, "---")
				}
				n, err := resolveNote(state, query)
				if err != nil {
					return fmt.Errorf("show: %w", err)
				}
				data, err := n.Marshal()
				if err != nil {
					return fmt.Errorf("show: marshal: %w", err)
				}
				fmt.Fprint(w, string(data))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&linkedFrom, "linked-from", "", "Show all notes linked from this ID")
	return cmd
}

// resolveNote finds a note by exact ID or case-insensitive title substring.
func resolveNote(state *rootState, query string) (*note.Note, error) {
	n, err := state.backend.Read(query)
	if err == nil {
		return n, nil
	}
	all, listErr := state.backend.List()
	if listErr != nil {
		return nil, fmt.Errorf("%w", err)
	}
	type match struct{ id, title string }
	var matches []match
	for _, candidate := range all {
		if strings.Contains(strings.ToLower(candidate.Title), strings.ToLower(query)) {
			matches = append(matches, match{candidate.ID, candidate.Title})
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no note found matching %q", query)
	case 1:
		return state.backend.Read(matches[0].id)
	default:
		return nil, fmt.Errorf("ambiguous query %q — %d matches; use full ID", query, len(matches))
	}
}
