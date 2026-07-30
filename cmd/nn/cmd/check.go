package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var representationSections = map[string][]string{
	"ontology": {"## Concepts", "## Relations"},
	"taxonomy": {"## Categories", "## Classification"},
	"axiom":    {"## Vocabulary", "## Invariant"},
}

func newCheckCmd(state *rootState) *cobra.Command {
	var (
		as                string
		setRepresentation bool
	)

	cmd := &cobra.Command{
		Use:   "check <id>",
		Short: "Validate a note's structural contract against its representation type",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			n, err := state.backend.Read(id)
			if err != nil {
				return fmt.Errorf("check: %w", err)
			}

			rep := as
			if rep == "" {
				rep = n.Representation
			}
			if rep == "" {
				return fmt.Errorf("check: note %s has no representation field; use --as <representation>", id)
			}

			required, ok := representationSections[rep]
			if !ok {
				return fmt.Errorf("check: unknown representation %q (known: ontology, taxonomy, axiom)", rep)
			}

			var missing []string
			for _, section := range required {
				if !strings.Contains(n.Body, section) {
					missing = append(missing, section)
				}
			}
			if len(missing) > 0 {
				return fmt.Errorf("check: %s fails %s validation — missing sections: %s", id, rep, strings.Join(missing, ", "))
			}

			if setRepresentation {
				n.Representation = rep
				n.Modified = time.Now()
				if err := state.backend.Update(n); err != nil {
					return fmt.Errorf("check: --set-representation: %w", err)
				}
				fmt.Fprintf(outWriter(cmd), "ok — set representation: %s on %s\n", rep, id)
				return nil
			}

			fmt.Fprintf(outWriter(cmd), "ok — %s passes %s validation\n", id, rep)
			return nil
		},
	}

	cmd.Flags().StringVar(&as, "as", "", "Override representation type for validation (ontology|taxonomy|axiom)")
	cmd.Flags().BoolVar(&setRepresentation, "set-representation", false, "Stamp representation field on note after passing validation")
	return cmd
}
