package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
)

// Compose the native selectors/renderers. No shell, alternate ownership resolver,
// alternate detector, or claim of an atomic snapshot across independently read streams.
func newTranscriptObserveCmd() *cobra.Command {
	var attention observeAttentionOptions
	command := &cobra.Command{
		Use:   "observe <session>",
		Short: "Read bounded streams and qualified attention signals",
		Long:  "One-shot readable observation: ROOT plus up to two canonical direct children, five ledger events each, 1000 readable characters per event. Canonical sampling is not recency or importance ranking. Independent reads; no monitor. Attention measures every bundled signal in an independent bounded cohort, retaining evidence and reporting unknown applicability explicitly. Output bounds do not bound source processing.",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if attention.Limit < 1 || attention.Limit > 20 {
				return fmt.Errorf("observe: --attention-limit requires 1..20")
			}
			if c.Flags().Changed("attention-agent") && c.Flags().Changed("attention-limit") {
				return fmt.Errorf("observe: --attention-agent cannot combine with --attention-limit")
			}
			if len(attention.Task) > 80 || (c.Flags().Changed("task") && strings.TrimSpace(attention.Task) == "") {
				return fmt.Errorf("observe: --task must be nonempty and at most 80 bytes")
			}
			run := func(command *cobra.Command, argv ...string) ([]byte, error) {
				var out bytes.Buffer
				command.SetOut(&out)
				command.SetErr(&out)
				command.SetArgs(argv)
				command.SilenceUsage = true
				command.SilenceErrors = true
				if err := command.ExecuteContext(c.Context()); err != nil {
					return nil, err
				}
				return out.Bytes(), nil
			}
			raw, err := run(newTranscriptTreeCmd(), args[0], "--parent", "ROOT", "--limit", "2", "--json")
			if err != nil {
				return err
			}
			var tree treeChildPage
			if err = json.Unmarshal(raw, &tree); err != nil {
				return err
			}
			var out bytes.Buffer
			fmt.Fprintf(&out, "Observation: %s\n", observeLabel(args[0]))
			fmt.Fprintf(&out, "sample: ROOT + %d of %d canonical direct children; omitted direct children: %d\n", tree.Returned, tree.TotalChildren, tree.Omitted)
			fmt.Fprintf(&out, "tree snapshot: %s\n", tree.Snapshot)
			fmt.Fprintln(&out, "Canonical order is not recency or importance. Other branches and older events remain uninspected.")
			fmt.Fprintln(&out, "Independent stream reads, not an atomic capture. Unavailable detail is unknown, not inactivity or success.")
			signals, err := observeAttentionSection(args[0], attention)
			if err != nil {
				return err
			}
			out.WriteString(signals)
			streams := []treeChildRow{{ID: "ROOT", Description: "orchestration stream"}}
			streams = append(streams, tree.Children...)
			for _, stream := range streams {
				raw, err = run(newTranscriptEventsCmd(), args[0], stream.ID, "--last", "5", "--format", "text", "--max-text-chars", "1000")
				if err != nil {
					return fmt.Errorf("observe %s: %w", stream.ID, err)
				}
				fmt.Fprintf(&out, "\n## %s — %s\n", observeLabel(stream.ID), observeLabel(stream.Description))
				out.Write(raw)
			}
			if out.Len() > 200000 {
				return fmt.Errorf("observe: output exceeds 200000 bytes; use targeted events reads")
			}
			_, err = c.OutOrStdout().Write(out.Bytes())
			return err
		},
	}
	command.Flags().StringVar(&attention.Task, "task", "", "optional cohort task override; attention runs without it")
	command.Flags().StringArrayVar(&attention.AgentTasks, "agent-task", nil, "optional per-agent task override ID=TASK for a selected attention agent")
	command.Flags().StringArrayVar(&attention.IDs, "attention-agent", nil, "exact attention agent ID; repeat up to 20 (independent of readable sample)")
	command.Flags().IntVar(&attention.Limit, "attention-limit", 20, "ROOT-first canonical attention population bound, 1..20; incompatible with explicit IDs")
	return command
}

func observeLabel(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
}
