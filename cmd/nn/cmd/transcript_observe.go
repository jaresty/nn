package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"
)

// Compose the native selectors/renderers. No shell, alternate ownership resolver,
// alternate detector, or claim of an atomic snapshot across independently read streams.
func newTranscriptObserveCmd() *cobra.Command {
	var attention observeAttentionOptions
	var recent time.Duration
	var refresh, snapshot string
	var coveragePage int
	command := &cobra.Command{
		Use:   "observe <session>",
		Short: "Read bounded streams and qualified attention signals",
		Long:  "One-shot readable observation: ROOT plus up to two canonical direct children, five ledger events each, 1000 readable characters per event. Canonical sampling is not recency or importance ranking. Independent reads; no monitor. Attention measures every bundled signal in an independent bounded cohort, retaining evidence and reporting unknown applicability explicitly. Output bounds do not bound source processing.",
		Args:  transcriptSessionArgs(cobra.ExactArgs(1)),
		RunE: func(c *cobra.Command, args []string) error {
			decodedBefore := transcriptDecodeCount.Load()
			if coveragePage < 0 {
				return fmt.Errorf("observe: --coverage-page must be positive")
			}
			if recent <= 0 {
				return fmt.Errorf("observe: --recent must be positive")
			}
			if snapshot != "" {
				for _, flag := range []string{"refresh", "recent", "task", "agent-task", "attention-agent", "attention-limit"} {
					if c.Flags().Changed(flag) {
						return fmt.Errorf("observe: replay cannot combine with --%s", flag)
					}
				}
				s, err := loadObserveState(snapshot, args[0])
				if err != nil {
					return err
				}
				text := observationText(s, snapshot)
				if coveragePage > 0 {
					text, err = renderObserveCoverage(s, coveragePage)
					if err != nil {
						return err
					}
				}
				_, err = fmt.Fprint(c.OutOrStdout(), text)
				return err
			}
			if c.Flags().Changed("coverage-page") {
				return fmt.Errorf("observe: --coverage-page requires --snapshot")
			}
			if refresh != "" && len(attention.IDs) > 0 {
				return fmt.Errorf("observe: conversation Refresh cannot combine with an agent restriction")
			}
			if refresh != "" {
				prior, err := loadObserveState(refresh, args[0])
				if err != nil {
					return err
				}
				if !c.Flags().Changed("recent") {
					recent = prior.Recent
				}
				if !c.Flags().Changed("task") {
					attention.Task = prior.Task
				}
				if !c.Flags().Changed("agent-task") {
					attention.AgentTasks = prior.AgentTasks
				}
			}
			if attention.Limit < 1 || attention.Limit > 20 {
				return fmt.Errorf("observe: --attention-limit requires 1..20")
			}
			if c.Flags().Changed("attention-agent") && c.Flags().Changed("attention-limit") {
				return fmt.Errorf("observe: --attention-agent cannot combine with --attention-limit")
			}
			if len(attention.Task) > 80 || (c.Flags().Changed("task") && strings.TrimSpace(attention.Task) == "") {
				return fmt.Errorf("observe: --task must be nonempty and at most 80 bytes")
			}
			if refresh != "" {
				retained, unchanged, e := tryObserveUnchanged(args[0], attention, recent, refresh)
				if e != nil {
					return e
				}
				if unchanged {
					decoded := transcriptDecodeCount.Load() - decodedBefore
					retained.DecodedRecords = &decoded
					id, e := saveObserveState(*retained)
					if e != nil {
						return e
					}
					text := observationText(*retained, id)
					if len(text) > 200000 {
						return fmt.Errorf("observe: output exceeds 200000 bytes")
					}
					_, e = fmt.Fprint(c.OutOrStdout(), text)
					return e
				}
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
			tree, parent, err := observeRoster(args[0])
			if err != nil {
				return err
			}
			var raw []byte
			var out bytes.Buffer
			fmt.Fprintf(&out, "Observation: %s\n", observeLabel(args[0]))
			fmt.Fprintf(&out, "sample: ROOT + %d of %d canonical direct children; omitted direct children: %d\n", tree.Returned, tree.TotalChildren, tree.Omitted)
			fmt.Fprintf(&out, "tree snapshot: %s\n", tree.Snapshot)
			if parent != nil {
				fmt.Fprintln(&out, "Roster: parent metadata only; worker usage not hydrated.")
			}
			fmt.Fprintln(&out, "Canonical order is not recency or importance. Other branches and older events remain uninspected.")
			fmt.Fprintln(&out, "Independent stream reads, not an atomic capture. Unavailable detail is unknown, not inactivity or success.")
			var signals string
			var live *observeLiveState
			if len(attention.IDs) > 0 {
				signals, err = observeAttentionSection(args[0], attention)
			} else {
				signals, live, err = observeLiveSection(args[0], attention, recent, refresh, parent)
			}
			if err != nil {
				return err
			}
			out.WriteString(signals)
			if live != nil {
				live.ReadableOffset = out.Len()
			}
			streams := []treeChildRow{{ID: "ROOT", Description: "orchestration stream"}}
			streams = append(streams, tree.Children...)
			for _, stream := range streams {
				if live != nil && stream.ID != "ROOT" {
					skipped := ""
					for _, a := range live.Agents {
						if a.ID == stream.ID && a.Fingerprint == "" && (a.State == "outside_window" || a.State == "deferred") {
							skipped = a.State
							break
						}
					}
					if skipped != "" {
						fmt.Fprintf(&out, "\n## %s — %s\nHistory not inspected: %s; use explicit agent events to inspect.\n", observeLabel(stream.ID), observeLabel(stream.Description), skipped)
						continue
					}
				}
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
			if live != nil {
				live.Text = out.String()
				decoded := transcriptDecodeCount.Load() - decodedBefore
				live.DecodedRecords = &decoded
				id, e := saveObserveState(*live)
				if e != nil {
					return e
				}
				out.Reset()
				out.WriteString(observationText(*live, id))
			}
			if out.Len() > 200000 {
				return fmt.Errorf("observe: output exceeds 200000 bytes; reduce --attention-limit")
			}
			_, err = c.OutOrStdout().Write(out.Bytes())
			return err
		},
	}
	command.Flags().StringVar(&attention.Task, "task", "", "optional cohort task override; attention runs without it")
	command.Flags().StringArrayVar(&attention.AgentTasks, "agent-task", nil, "optional per-agent task override ID=TASK for a selected attention agent")
	command.Flags().StringArrayVar(&attention.IDs, "attention-agent", nil, "exact attention agent ID; repeat up to 20 (independent of readable sample)")
	command.Flags().IntVar(&attention.Limit, "attention-limit", 20, "attention detail display limit, 1..20; does not limit conversation evaluation")
	command.Flags().DurationVar(&recent, "recent", 5*time.Minute, "initial evidence recency window; unknown recency is included")
	command.Flags().StringVar(&refresh, "refresh", "", "evaluate new/changed agents against a retained observation snapshot")
	command.Flags().StringVar(&snapshot, "snapshot", "", "replay a retained observation without source acquisition")
	command.Flags().IntVar(&coveragePage, "coverage-page", 0, "retained coverage page of 20 agents (requires --snapshot)")
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
