package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/rules"
)

// buildRuleEngine loads all notes, derives facts, and loads the built-in rules
// plus any user rules embedded in note bodies via ```nn-rule fences. It returns
// the evaluated engine and any per-note parse warnings (which never abort load).
func buildRuleEngine(state *rootState, assume ...string) (*rules.Engine, []string, error) {
	all, err := state.backend.List()
	if err != nil {
		return nil, nil, fmt.Errorf("rules: %w", err)
	}

	e := rules.NewEngine()
	for _, f := range rules.FactsFromNotes(all) {
		e.AddFact(f)
	}

	// Hypothetical/counterfactual facts (--assume): injected alongside the real
	// facts before Eval so the ruleset is evaluated AS IF they were present.
	// They are never written to disk, and because the engine is rebuilt per
	// invocation they are discarded when the command returns.
	for _, a := range assume {
		f, err := parseFactArg(a)
		if err != nil {
			return nil, nil, fmt.Errorf("rules: --assume %q: %w", a, err)
		}
		e.AddFact(f)
	}

	// Built-in invariants.
	builtin, err := rules.ParseProgram(rules.BuiltinRules())
	if err != nil {
		return nil, nil, fmt.Errorf("rules: builtin: %w", err)
	}
	e.AddRules(builtin)

	// User rules embedded in note bodies.
	var warns []string
	for _, n := range all {
		userRules, w := rules.ExtractFenceRules(n.ID, n.Body)
		e.AddRules(userRules)
		warns = append(warns, w...)
	}

	if err := e.Eval(); err != nil {
		return nil, warns, fmt.Errorf("rules: %w", err)
	}
	return e, warns, nil
}

func newRulesCmd(state *rootState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Evaluate Datalog rules embedded in notes (validation & derivation)",
	}
	cmd.AddCommand(newRulesCheckCmd(state), newRulesQueryCmd(state), newRulesListCmd(state), newRulesExplainCmd(state))
	return cmd
}

// newRulesExplainCmd prints the derivation path of a derived fact.
func newRulesExplainCmd(state *rootState) *cobra.Command {
	return &cobra.Command{
		Use:   "explain <fact>",
		Short: "Show how a derived fact was produced, down to base facts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fact, err := parseFactArg(args[0])
			if err != nil {
				return err
			}
			e, warns, err := buildRuleEngine(state)
			if err != nil {
				return err
			}
			printWarns(cmd, warns)

			steps, ok := e.Explain(fact)
			if !ok {
				return fmt.Errorf("rules explain: fact %q was not derived", args[0])
			}
			out := outWriter(cmd)
			for _, s := range steps {
				fmt.Fprintln(out, s)
			}
			return nil
		},
	}
}

// newRulesCheckCmd runs the engine and prints every violation(ID, Reason) fact,
// exiting non-zero if any exist.
func newRulesCheckCmd(state *rootState) *cobra.Command {
	var assume []string
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Report rule violations across all notes",
		Long: "Report rule violations across all notes.\n\n" +
			"--assume injects a hypothetical ground fact (e.g. " +
			"representation(ID, ontology)) so the ruleset is evaluated as if that " +
			"fact were present, without writing anything to disk. Repeatable.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			e, warns, err := buildRuleEngine(state, assume...)
			if err != nil {
				return err
			}
			printWarns(cmd, warns)

			viols := e.Query("violation")
			out := outWriter(cmd)
			if len(viols) == 0 {
				fmt.Fprintln(out, "ok — no rule violations")
				return nil
			}
			for _, v := range viols {
				id, reason := violationParts(v)
				fmt.Fprintf(out, "%s: %s\n", id, reason)
			}
			return fmt.Errorf("rules check: %d violation(s)", len(viols))
		},
	}
	cmd.Flags().StringArrayVar(&assume, "assume", nil,
		"assume a hypothetical fact, e.g. representation(ID, ontology) (repeatable)")
	return cmd
}

// newRulesQueryCmd prints all derived facts for a predicate. The argument is a
// predicate name or a pattern like "PRED(...)" (only the predicate name is used
// to select facts; argument patterns are not yet filtered).
func newRulesQueryCmd(state *rootState) *cobra.Command {
	return &cobra.Command{
		Use:   "query <predicate>",
		Short: "Print derived facts for a predicate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pred := predicateName(args[0])
			e, warns, err := buildRuleEngine(state)
			if err != nil {
				return err
			}
			printWarns(cmd, warns)

			facts := e.Query(pred)
			out := outWriter(cmd)
			if len(facts) == 0 {
				fmt.Fprintf(out, "no facts for %q\n", pred)
				return nil
			}
			for _, f := range facts {
				fmt.Fprintln(out, f.Key())
			}
			return nil
		},
	}
}

// newRulesListCmd lists the rules loaded from note bodies, with note-ID
// provenance, plus the count of built-in rules.
func newRulesListCmd(state *rootState) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List loaded rules and their provenance",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			all, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("rules: %w", err)
			}
			out := outWriter(cmd)

			builtin, err := rules.ParseProgram(rules.BuiltinRules())
			if err != nil {
				return fmt.Errorf("rules: builtin: %w", err)
			}
			fmt.Fprintf(out, "built-in: %d rule(s)\n", len(builtin))

			type entry struct {
				id    string
				count int
			}
			var entries []entry
			var warns []string
			for _, n := range all {
				userRules, w := rules.ExtractFenceRules(n.ID, n.Body)
				warns = append(warns, w...)
				if len(userRules) > 0 {
					entries = append(entries, entry{n.ID, len(userRules)})
				}
			}
			sort.Slice(entries, func(i, j int) bool { return entries[i].id < entries[j].id })
			for _, e := range entries {
				fmt.Fprintf(out, "%s: %d rule(s)\n", e.id, e.count)
			}
			printWarns(cmd, warns)
			return nil
		},
	}
}

// ── helpers ────────────────────────────────────────────────────────────────────

func printWarns(cmd *cobra.Command, warns []string) {
	for _, w := range warns {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w)
	}
}

// violationParts splits a violation(ID, Reason) fact into its two arguments.
func violationParts(f rules.Fact) (id, reason string) {
	if len(f.Args) >= 2 {
		return f.Args[0], f.Args[1]
	}
	if len(f.Args) == 1 {
		return f.Args[0], ""
	}
	return "", ""
}

// predicateName extracts the predicate name from "pred" or "pred(...)".
func predicateName(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '('); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

// parseFactArg parses a ground fact literal "pred(a, b, c)" into a rules.Fact.
func parseFactArg(s string) (rules.Fact, error) {
	s = strings.TrimSpace(s)
	open := strings.IndexByte(s, '(')
	if open < 0 || !strings.HasSuffix(s, ")") {
		return rules.Fact{}, fmt.Errorf("expected a fact of the form pred(arg,...), got %q", s)
	}
	pred := strings.TrimSpace(s[:open])
	inner := s[open+1 : len(s)-1]
	f := rules.Fact{Pred: pred}
	for _, part := range strings.Split(inner, ",") {
		f.Args = append(f.Args, strings.Trim(strings.TrimSpace(part), `"'`))
	}
	return f, nil
}
