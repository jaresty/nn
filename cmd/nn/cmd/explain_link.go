package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newExplainLinkCmd(state *rootState) *cobra.Command {
	var (
		linkType string
		asJSON   bool
	)

	cmd := &cobra.Command{
		Use:   "explain-link <from-id> <to-id>",
		Short: "Print the semantic definition of a link type as applied between two notes",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if linkType == "" {
				return fmt.Errorf("--type is required")
			}
			def, ok := note.LinkTypeDescriptions[linkType]
			if !ok {
				return fmt.Errorf("explain-link: unknown link type %q (known: %s)", linkType, strings.Join(note.LinkTypeOrder, ", "))
			}

			from, err := state.backend.Read(args[0])
			if err != nil {
				return fmt.Errorf("explain-link: from note: %w", err)
			}
			to, err := state.backend.Read(args[1])
			if err != nil {
				return fmt.Errorf("explain-link: to note: %w", err)
			}

			warning := note.LinkTypeWarnings[linkType]

			if asJSON {
				out := map[string]any{
					"type":       linkType,
					"definition": def,
					"from":       map[string]string{"id": from.ID, "title": from.Title},
					"to":         map[string]string{"id": to.ID, "title": to.Title},
					"reads_as":   fmt.Sprintf("%q %s %q", from.Title, linkType, to.Title),
				}
				if warning != "" {
					out["warning"] = warning
				}
				enc, _ := json.MarshalIndent(out, "", "  ")
				fmt.Fprintln(outWriter(cmd), string(enc))
				return nil
			}

			w := outWriter(cmd)
			fmt.Fprintf(w, "Link type: %s\n\n", linkType)
			fmt.Fprintf(w, "Definition: %s\n\n", def)
			fmt.Fprintf(w, "As applied:\n  %q  -[%s]->  %q\n\n", from.Title, linkType, to.Title)
			fmt.Fprintf(w, "Reads as: \"%s\" %s \"%s\"\n", from.Title, linkType, to.Title)
			if warning != "" {
				fmt.Fprintf(w, "\n⚠  Warning: %s\n", warning)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&linkType, "type", "", "Link type to explain (required)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")
	return cmd
}
