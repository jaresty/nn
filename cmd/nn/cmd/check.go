package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

// taxonomyLinkTypes lists the only link types permitted in a taxonomy subgraph.
var taxonomyLinkTypes = map[string]bool{
	"refines": true,
	"extends": true,
}

type checkViolation struct {
	nodeID  string
	message string
}

// checkRepresentationGraph traverses the representation subgraph rooted at root,
// following only links whose targets share the same representation value.
// It returns all violations found across the subgraph.
func checkRepresentationGraph(root *note.Note, byID map[string]*note.Note, rep string) []checkViolation {
	var violations []checkViolation

	// BFS with cycle detection.
	visited := map[string]bool{}
	inStack := map[string]bool{}
	var dfs func(n *note.Note, isRoot bool)
	dfs = func(n *note.Note, isRoot bool) {
		if inStack[n.ID] {
			violations = append(violations, checkViolation{n.ID, fmt.Sprintf("cycle detected at %s (%s)", n.ID, n.Title)})
			return
		}
		if visited[n.ID] {
			return
		}
		visited[n.ID] = true
		inStack[n.ID] = true
		defer func() { inStack[n.ID] = false }()

		// Node type check.
		if isRoot {
			if n.Type != note.TypeModel {
				violations = append(violations, checkViolation{n.ID, fmt.Sprintf("root %s must be type:model, got type:%s", n.ID, n.Type)})
			}
		} else {
			if n.Type != note.TypeConcept && n.Type != note.TypeArgument {
				violations = append(violations, checkViolation{n.ID, fmt.Sprintf("non-root %s must be type:concept or type:argument, got type:%s", n.ID, n.Type)})
			}
		}

		// Collect same-representation outgoing links.
		var sameRepLinks []note.Link
		for _, lnk := range n.Links {
			target, ok := byID[lnk.TargetID]
			if !ok || target.Representation != rep {
				continue
			}
			sameRepLinks = append(sameRepLinks, lnk)
		}

		// Representation-specific link type checks.
		if rep == "taxonomy" {
			for _, lnk := range sameRepLinks {
				if !taxonomyLinkTypes[lnk.Type] {
					violations = append(violations, checkViolation{n.ID, fmt.Sprintf("taxonomy node %s has disallowed link type %q (only refines/extends permitted)", n.ID, lnk.Type)})
				}
			}
		}
		if rep == "axiom" && isRoot {
			hasGroundedBy := false
			for _, lnk := range sameRepLinks {
				if lnk.Type == "grounded-by" {
					hasGroundedBy = true
					break
				}
			}
			if !hasGroundedBy {
				violations = append(violations, checkViolation{n.ID, fmt.Sprintf("axiom root %s must have at least one grounded-by link within the same-representation subgraph", n.ID)})
			}
		}

		for _, lnk := range sameRepLinks {
			target := byID[lnk.TargetID]
			dfs(target, false)
		}
	}

	dfs(root, true)
	return violations
}

// runRepresentationCheck validates n's representation subgraph and prints the result to cmd's output.
// It loads all notes from state to build the ID map. Returns an error if validation fails.
func runRepresentationCheck(cmd *cobra.Command, state *rootState, n *note.Note) error {
	all, err := state.backend.List()
	if err != nil {
		return fmt.Errorf("check: %w", err)
	}
	byID := make(map[string]*note.Note, len(all))
	for _, nn := range all {
		byID[nn.ID] = nn
	}
	violations := checkRepresentationGraph(n, byID, n.Representation)
	if len(violations) > 0 {
		var msgs []string
		for _, v := range violations {
			msgs = append(msgs, v.message)
		}
		return fmt.Errorf("check: %s fails %s graph validation:\n  %s", n.ID, n.Representation, strings.Join(msgs, "\n  "))
	}
	fmt.Fprintf(outWriter(cmd), "ok — %s passes %s graph validation\n", n.ID, n.Representation)
	return nil
}

func newCheckCmd(state *rootState) *cobra.Command {
	var (
		as                string
		setRepresentation bool
	)

	cmd := &cobra.Command{
		Use:   "check <id>",
		Short: "Validate a note's representation subgraph structure",
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

			validReps := map[string]bool{"ontology": true, "taxonomy": true, "axiom": true}
			if !validReps[rep] {
				return fmt.Errorf("check: unknown representation %q (known: ontology, taxonomy, axiom)", rep)
			}

			// Load all notes to build the ID lookup map.
			all, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("check: %w", err)
			}
			byID := make(map[string]*note.Note, len(all))
			for _, nn := range all {
				byID[nn.ID] = nn
			}

			violations := checkRepresentationGraph(n, byID, rep)
			if len(violations) > 0 {
				var msgs []string
				for _, v := range violations {
					msgs = append(msgs, v.message)
				}
				return fmt.Errorf("check: %s fails %s graph validation:\n  %s", id, rep, strings.Join(msgs, "\n  "))
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

			fmt.Fprintf(outWriter(cmd), "ok — %s passes %s graph validation\n", id, rep)
			return nil
		},
	}

	cmd.Flags().StringVar(&as, "as", "", "Override representation type for validation (ontology|taxonomy|axiom)")
	cmd.Flags().BoolVar(&setRepresentation, "set-representation", false, "Stamp representation field on note after passing validation")
	return cmd
}
