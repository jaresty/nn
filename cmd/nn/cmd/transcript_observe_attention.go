package cmd

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

type observeAttentionOptions struct {
	IDs        []string
	Limit      int
	Task       string
	AgentTasks []string
}

func selectObserveAttention(agents []agent, o observeAttentionOptions) ([]string, error) {
	known := map[string]bool{}
	for _, a := range agents {
		known[a.ID] = true
	}
	if len(o.IDs) > 0 {
		if len(o.IDs) > 20 {
			return nil, fmt.Errorf("observe: attention requires at most 20 explicit IDs")
		}
		seen := map[string]bool{}
		for _, id := range o.IDs {
			if id == "" || seen[id] {
				return nil, fmt.Errorf("observe: empty or duplicate attention agent ID")
			}
			if !known[id] {
				return nil, fmt.Errorf("observe: unknown attention agent %q", id)
			}
			seen[id] = true
		}
		return append([]string(nil), o.IDs...), nil
	}
	if o.Limit < 1 || o.Limit > 20 {
		return nil, fmt.Errorf("observe: --attention-limit requires 1..20")
	}
	ids := []string{}
	for id := range known {
		if id != "ROOT" {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if known["ROOT"] {
		ids = append([]string{"ROOT"}, ids...)
	}
	if len(ids) > o.Limit {
		ids = ids[:o.Limit]
	}
	return ids, nil
}

func observeAttentionSection(session string, o observeAttentionOptions) (string, error) {
	agents, err := buildTree(session)
	if err != nil {
		return "", err
	}
	ids, err := selectObserveAttention(agents, o)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	fmt.Fprintln(&out, "\nAttention signals")
	mode := "ROOT-first canonical population (includes descendants); not recency or importance"
	if len(o.IDs) > 0 {
		mode = "explicit exact IDs in requested order"
	}
	fmt.Fprintf(&out, "Attention selection: %s; independent of the readable two-child sample\n", mode)
	fmt.Fprintf(&out, "Candidate inventory: selected %d of %d; outside attention cohort: %d\n", len(ids), len(agents), len(agents)-len(ids))
	fmt.Fprintf(&out, "Selected attention IDs: %s\n", observeLabel(strings.Join(ids, ", ")))
	overrides, err := parseAgentTasks(o.AgentTasks)
	if err != nil {
		return "", err
	}
	for id := range overrides {
		found := false
		for _, selected := range ids {
			if id == selected {
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("observe: agent-task names unselected agent %q", id)
		}
	}
	fmt.Fprintln(&out, "All bundled signals are measured; task overrides affect applicability, not activation.")
	retained, err := buildAttentionWithTasks(session, ids, o.Task, overrides)
	if err != nil {
		fmt.Fprintf(&out, "attention error — batch did not complete; no successful evaluation published: %s\n", observeLabel(err.Error()))
		fmt.Fprintln(&out, "Do not interpret this error as no findings. No retry or scope substitution performed.")
		return out.String(), nil
	}
	page := retained.Page
	fmt.Fprintf(&out, "Metric version: %d. Retained evaluation scope follows; candidate inventory and evaluation are independent acquisitions.\n", page.MetricVersion)
	if err = renderAttention(&out, page); err != nil {
		return "", err
	}
	return out.String(), nil
}
