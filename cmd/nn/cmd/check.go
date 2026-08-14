package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
	"github.com/jaresty/nn/internal/rules"
)

// engineCheckViolations computes representation-subgraph violations via the
// rules engine (builtin.dl), scoped to the same-representation subgraph rooted
// at root. It returns the same []checkViolation shape as the legacy
// checkRepresentationGraph so nn check can delegate to it. The differential
// guard in rules_differential_test.go proves the two agree on verdicts.
func engineCheckViolations(root *note.Note, all []*note.Note, rep string) []checkViolation {
	// Scope: root plus every node reachable from root via same-representation
	// links (mirrors checkRepresentationGraph's same-rep traversal). The root is
	// always in scope even if its representation field is empty (e.g. `--as`).
	inScope := sameRepSubgraph(root, all, rep)

	e := rules.NewEngine()
	for _, f := range rules.FactsFromNotes(all) {
		e.AddFact(f)
	}
	// Ensure every in-scope node carries the representation being validated, so
	// the built-in rep_* rules fire even when the note's representation field is
	// absent (the `--as <rep>` override supplies rep as an argument, mirroring
	// checkRepresentationGraph which validates the root regardless of its field).
	for id := range inScope {
		e.AddFact(rules.Fact{Pred: "representation", Args: []string{id, rep}})
	}
	builtin, err := rules.ParseProgram(rules.BuiltinRules())
	if err != nil {
		// builtin.dl is embedded and tested; a parse failure is a programmer error.
		return []checkViolation{{root.ID, fmt.Sprintf("internal: builtin rules failed to parse: %v", err)}}
	}
	e.AddRules(builtin)
	if err := e.Eval(); err != nil {
		return []checkViolation{{root.ID, fmt.Sprintf("internal: rule evaluation failed: %v", err)}}
	}

	var out []checkViolation
	for _, f := range e.Query("violation") {
		if len(f.Args) < 2 {
			continue
		}
		id, reason := f.Args[0], f.Args[1]
		if inScope[id] {
			out = append(out, checkViolation{id, reason})
		}
	}
	return out
}

// sameRepSubgraph returns the set of note IDs in the same-representation
// subgraph rooted at root (root + all nodes reachable via links whose both
// endpoints have representation rep).
func sameRepSubgraph(root *note.Note, all []*note.Note, rep string) map[string]bool {
	byID := make(map[string]*note.Note, len(all))
	for _, n := range all {
		byID[n.ID] = n
	}
	inScope := map[string]bool{}
	var walk func(n *note.Note)
	walk = func(n *note.Note) {
		if inScope[n.ID] {
			return
		}
		inScope[n.ID] = true
		for _, lnk := range n.Links {
			target, ok := byID[lnk.TargetID]
			if !ok || target.Representation != rep {
				continue
			}
			walk(target)
		}
	}
	walk(root)
	return inScope
}

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
	violations := engineCheckViolations(n, all, n.Representation)
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

// findRepresentationRoot traverses same-representation inbound links from start,
// walking toward a type=model node. In representation subgraphs, the root links
// TO children (root.Links contains children), so to reach root from a child we
// follow inbound links — notes that have a same-rep link whose target is start.
func findRepresentationRoot(start *note.Note, byID map[string]*note.Note, inbound map[string][]*note.Note, rep string) (*note.Note, error) {
	visited := map[string]bool{}
	cur := start
	for {
		if visited[cur.ID] {
			return nil, fmt.Errorf("check: cycle detected — cannot resolve root from %s", start.ID)
		}
		visited[cur.ID] = true

		// A representation node has at most one same-representation parent. A model
		// is a root only when it has none; an inbound parent would make it an
		// intermediate model or part of a cycle, neither of which is a unique root.
		parents := make(map[string]*note.Note)
		for _, parent := range inbound[cur.ID] {
			if parent.Representation == rep {
				parents[parent.ID] = parent
			}
		}
		if cur.Type == note.TypeModel {
			if len(parents) == 0 {
				return cur, nil
			}
			return nil, fmt.Errorf("check: type:model %s has same-representation inbound ancestry", cur.ID)
		}
		if len(parents) == 0 {
			return nil, fmt.Errorf("check: no type:model root reachable via same-representation inbound links from %s", start.ID)
		}
		if len(parents) > 1 {
			return nil, fmt.Errorf("check: ambiguous representation ancestry at %s — %d same-representation parents", cur.ID, len(parents))
		}
		for _, parent := range parents {
			cur = parent
		}
	}
}

func newCheckCmd(state *rootState) *cobra.Command {
	var (
		as                string
		setRepresentation bool
		root              string
	)

	cmd := &cobra.Command{
		Use:   "check <id>",
		Short: "Validate a note's representation subgraph structure",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if root != "" && root != "auto" {
				return fmt.Errorf("check: --root only accepts \"auto\" (got %q)", root)
			}

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

			// Load all notes to build the ID lookup map and inbound map.
			all, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("check: %w", err)
			}
			byID := make(map[string]*note.Note, len(all))
			for _, nn := range all {
				byID[nn.ID] = nn
			}

			if root == "auto" {
				inboundMap := make(map[string][]*note.Note, len(all))
				for _, nn := range all {
					for _, lnk := range nn.Links {
						inboundMap[lnk.TargetID] = append(inboundMap[lnk.TargetID], nn)
					}
				}
				found, findErr := findRepresentationRoot(n, byID, inboundMap, rep)
				if findErr != nil {
					return findErr
				}
				n = found
				id = found.ID
			}

			violations := engineCheckViolations(n, all, rep)
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
				if err := state.backend.Update(n, nil); err != nil {
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
	cmd.Flags().StringVar(&root, "root", "", "Resolve representation root automatically via backlinks before validating (use: --root auto)")
	return cmd
}
