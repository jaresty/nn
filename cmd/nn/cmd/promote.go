package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newPromoteCmd(state *rootState) *cobra.Command {
	var (
		to       string
		subgraph string
		ifValid  bool
	)

	cmd := &cobra.Command{
		Use:   "promote <id>",
		Short: "Advance note status: draft → reviewed → permanent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if to == "" {
				return fmt.Errorf("--to is required (reviewed|permanent)")
			}
			status := note.Status(to)
			if !status.IsValid() {
				return fmt.Errorf("invalid --to %q: must be reviewed|permanent", to)
			}

			if subgraph != "" {
				return promoteSubgraph(cmd, state, subgraph, status, ifValid)
			}

			n, err := resolveNote(state, args[0])
			if err != nil {
				return fmt.Errorf("promote: %w", err)
			}
			if err := state.backend.Promote(n.ID, n.Modified, status); err != nil {
				return fmt.Errorf("promote: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "promoted %s to %s\n", n.ID, to)
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "Target status: reviewed|permanent")
	cmd.Flags().StringVar(&subgraph, "subgraph", "", "Promote all notes in the representation subgraph rooted at this ID")
	cmd.Flags().BoolVar(&ifValid, "if-valid", false, "Run representation graph validation before promoting; abort on violations")
	return cmd
}

// promoteSubgraph promotes all notes reachable via same-representation outgoing links
// from rootID, in leaves-first order (post-order DFS). If ifValid is true, validates
// the subgraph first and returns all violations without promoting.
func promoteSubgraph(cmd *cobra.Command, state *rootState, rootID string, status note.Status, ifValid bool) error {
	all, err := state.backend.List()
	if err != nil {
		return fmt.Errorf("promote --subgraph: %w", err)
	}
	byID := make(map[string]*note.Note, len(all))
	for _, n := range all {
		byID[n.ID] = n
	}

	root, ok := byID[rootID]
	if !ok {
		return fmt.Errorf("promote --subgraph: root note %s not found", rootID)
	}

	rep := root.Representation
	if rep == "" {
		return fmt.Errorf("promote --subgraph: root note %s has no representation field", rootID)
	}

	if ifValid {
		violations := checkRepresentationGraph(root, byID, rep)
		if len(violations) > 0 {
			var msgs []string
			for _, v := range violations {
				msgs = append(msgs, v.message)
			}
			return fmt.Errorf("promote --subgraph: %s fails %s graph validation:\n  %s", rootID, rep, strings.Join(msgs, "\n  "))
		}
	}

	// Collect nodes in post-order (leaves first) via DFS.
	visited := map[string]bool{}
	var order []*note.Note
	var collect func(n *note.Note)
	collect = func(n *note.Note) {
		if visited[n.ID] {
			return
		}
		visited[n.ID] = true
		for _, lnk := range n.Links {
			target, ok := byID[lnk.TargetID]
			if !ok || target.Representation != rep {
				continue
			}
			collect(target)
		}
		order = append(order, n)
	}
	collect(root)

	now := time.Now()
	w := outWriter(cmd)
	for _, n := range order {
		if err := state.backend.Promote(n.ID, n.Modified, status); err != nil {
			return fmt.Errorf("promote --subgraph: %s: %w", n.ID, err)
		}
		n.Modified = now
		fmt.Fprintf(w, "promoted %s to %s\n", n.ID, status)
	}
	return nil
}
