package cmd

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type ledgerWindowOptions struct {
	Enabled    bool     `json:"-"`
	AnchorIDs  []string `json:"anchor_event_ids,omitempty"`
	Kind       string   `json:"kind"`
	Role       string   `json:"role"`
	Search     string   `json:"search"`
	Regex      bool     `json:"regex"`
	Before     int      `json:"before_context"`
	After      int      `json:"after_context"`
	Context    int      `json:"-"`
	MaxMatches int      `json:"max_matches"`
}

type ledgerWindowReceipt struct {
	ledgerWindowOptions
	Anchor          string `json:"anchor_event,omitempty"`
	Matches         int    `json:"matching_events"`
	SelectedMatches int    `json:"selected_matches"`
	ContextEvents   int    `json:"context_events"`
	OmittedMatches  int    `json:"omitted_matches"`
	Windows         int    `json:"windows"`
}

func (o *ledgerWindowOptions) flags(c *cobra.Command) {
	c.Flags().StringVar(&o.Kind, "kind", "", "match message, tool_call, tool_result, or lifecycle events")
	c.Flags().StringVar(&o.Role, "role", "", "match message events with this role (e.g. assistant or user)")
	c.Flags().StringVar(&o.Search, "search", "", "case-insensitive literal search of readable event content and tool arguments")
	c.Flags().BoolVar(&o.Regex, "regex", false, "interpret --search as a Go regular expression (case-sensitive by default)")
	c.Flags().IntVarP(&o.Before, "before-context", "B", 0, "include N preceding ledger events per match (0..200)")
	c.Flags().IntVarP(&o.After, "after-context", "A", 0, "include N following ledger events per match (0..200)")
	c.Flags().IntVarP(&o.Context, "context", "C", 0, "include N ledger events on each side; incompatible with -A/-B")
	c.Flags().IntVar(&o.MaxMatches, "max-matches", 20, "maximum matching anchors before context expansion (1..200; filtered queries)")
}

func (o *ledgerWindowOptions) configure(c *cobra.Command, event, format string) error {
	flags := []string{"kind", "role", "search", "regex", "before-context", "after-context", "context", "max-matches"}
	for _, f := range flags {
		if c.Flags().Changed(f) {
			o.Enabled = true
		}
	}
	if event != "" && format == "text" {
		o.Enabled = true
	}
	if !o.Enabled {
		return nil
	}
	for _, f := range []string{"summary", "at"} {
		if c.Flags().Changed(f) {
			return fmt.Errorf("events: search/context cannot combine with --%s", f)
		}
	}
	if c.Flags().Changed("context") {
		if c.Flags().Changed("before-context") || c.Flags().Changed("after-context") {
			return fmt.Errorf("events: --context cannot combine with --before-context or --after-context")
		}
		o.Before, o.After = o.Context, o.Context
	}
	if o.Before < 0 || o.Before > 200 || o.After < 0 || o.After > 200 || o.MaxMatches < 1 || o.MaxMatches > 200 {
		return fmt.Errorf("events: context requires 0..200 and --max-matches requires 1..200")
	}
	if c.Flags().Changed("kind") {
		switch o.Kind {
		case "message", "tool_call", "tool_result", "lifecycle":
		default:
			return fmt.Errorf("events: invalid --kind %q", o.Kind)
		}
	}
	if c.Flags().Changed("role") && strings.TrimSpace(o.Role) == "" {
		return fmt.Errorf("events: --role must not be empty")
	}
	if c.Flags().Changed("search") && o.Search == "" {
		return fmt.Errorf("events: --search must not be empty")
	}
	if c.Flags().Changed("regex") && !c.Flags().Changed("search") {
		return fmt.Errorf("events: --regex requires --search")
	}
	if o.Regex {
		if _, err := regexp.Compile(o.Search); err != nil {
			return fmt.Errorf("events: invalid search regex: %w", err)
		}
	}
	if event != "" {
		for _, f := range []string{"kind", "role", "search", "regex", "since", "until", "errors-only", "last", "max-matches"} {
			if c.Flags().Changed(f) {
				return fmt.Errorf("events: --event cannot combine with --%s", f)
			}
		}
	}
	return nil
}

// Search only semantic display fields. Never serialize the whole message: it can
// contain reasoning, signatures, usage, or other unrelated opaque metadata.
func ledgerSearchText(e ledgerEvent) string {
	raw, _ := json.Marshal(e["payload"])
	var p map[string]json.RawMessage
	_ = json.Unmarshal(raw, &p)
	switch e["kind"] {
	case "message", "tool_result":
		return ledgerText(p["content"])
	case "tool_call":
		args := p["arguments"]
		if len(args) == 0 {
			args = p["input"]
		}
		// Decode strings so a literal search sees the actual command, not JSON escapes.
		var value any
		if json.Unmarshal(args, &value) != nil {
			return ledgerString(p, "name")
		}
		return ledgerString(p, "name") + " " + ledgerArgumentText(value)
	case "lifecycle":
		return ledgerString(p, "status")
	}
	return ""
}

func ledgerArgumentText(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []any:
		parts := make([]string, len(x))
		for i, item := range x {
			parts[i] = ledgerArgumentText(item)
		}
		return strings.Join(parts, " ")
	default:
		// JSON canonicalizes object key order; string escaping in objects is decoded
		// below through recursion so matching is independent of encoding spelling.
		if m, ok := v.(map[string]any); ok {
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			parts := []string{}
			for _, k := range keys {
				parts = append(parts, k, ledgerArgumentText(m[k]))
			}
			return strings.Join(parts, " ")
		}
		b, _ := json.Marshal(x)
		return string(b)
	}
}

func buildWindowLedgerPage(session, id, schema, detail string, selection []string, payload bool, events []ledgerEvent, page int, supplied, event string, all bool, q ledgerQuery, o ledgerWindowOptions) (ledgerPage, error) {
	// Bind all evidence used for matching, even when callers hide payload/facets.
	full, err := buildLedgerPage(session, id, schema, detail, []string{"identity", "message", "usage", "tools", "lifecycle"}, true, events, 1, "", "", true)
	if err != nil {
		return ledgerPage{}, err
	}
	receipt := &ledgerQueryReceipt{ErrorsOnly: q.ErrorsOnly, Clock: "record_preferred_message_fallback", Boundary: "inclusive anchors; context ignores anchor filters", LedgerSnapshot: full.Snapshot, Total: len(events), Completeness: "Complete selected projection only after every page/segment; not original-source completeness."}
	if q.Since != nil {
		s := q.Since.UTC().Format(time.RFC3339Nano)
		receipt.Since = &s
	}
	if q.Until != nil {
		s := q.Until.UTC().Format(time.RFC3339Nano)
		receipt.Until = &s
	}
	var pattern *regexp.Regexp
	if o.Regex {
		pattern, err = regexp.Compile(o.Search)
		if err != nil {
			return ledgerPage{}, err
		}
	}
	matches := []int{}
	explicit := map[string]bool{}
	for _, anchor := range o.AnchorIDs {
		explicit[anchor] = true
	}
	for i, e := range events {
		if len(explicit) > 0 {
			eid, _ := e["event_id"].(string)
			if !explicit[eid] {
				continue
			}
		}
		if event != "" && e["event_id"] != event {
			continue
		}
		if q.Since != nil || q.Until != nil {
			ts, status := ledgerTime(e["timestamp"])
			if status != "known" {
				receipt.Unknown++
				continue
			}
			if q.Since != nil && ts.Before(*q.Since) {
				receipt.Before++
				continue
			}
			if q.Until != nil && ts.After(*q.Until) {
				receipt.After++
				continue
			}
		}
		if q.ErrorsOnly && !ledgerEventError(e) {
			receipt.NonErrors++
			continue
		}
		if o.Kind != "" && e["kind"] != o.Kind {
			continue
		}
		if o.Role != "" {
			m, _ := e["message"].(map[string]any)
			if e["kind"] != "message" || m["role"] != o.Role {
				continue
			}
		}
		if o.Search != "" {
			text := ledgerSearchText(e)
			if pattern != nil {
				if !pattern.MatchString(text) {
					continue
				}
			} else if !strings.Contains(strings.ToLower(text), strings.ToLower(o.Search)) {
				continue
			}
		}
		matches = append(matches, i)
	}
	if len(explicit) > 0 && len(matches) != len(explicit) {
		return ledgerPage{}, fmt.Errorf("events: context anchor unavailable or source changed")
	}
	if event != "" && len(matches) != 1 {
		return ledgerPage{}, fmt.Errorf("events: event id not found")
	}
	wr := &ledgerWindowReceipt{ledgerWindowOptions: o, Anchor: event, Matches: len(matches)}
	if q.Last > 0 {
		receipt.RequestedLast = &q.Last
		n := q.Last
		if n > o.MaxMatches {
			n = o.MaxMatches
		}
		if len(matches) > n {
			matches = matches[len(matches)-n:]
		}
	} else if len(matches) > o.MaxMatches {
		matches = matches[:o.MaxMatches]
	}
	wr.SelectedMatches = len(matches)
	wr.OmittedMatches = wr.Matches - len(matches)
	if q.Last > 0 {
		receipt.MatchingEvents = &wr.Matches
		receipt.ReturnedEvents = &wr.SelectedMatches
		older := wr.OmittedMatches > 0
		receipt.OlderMatching = &older
	}
	anchors := map[int]bool{}
	included := map[int]bool{}
	for _, i := range matches {
		anchors[i] = true
		low, high := i-o.Before, i+o.After
		if low < 0 {
			low = 0
		}
		if high >= len(events) {
			high = len(events) - 1
		}
		for j := low; j <= high; j++ {
			included[j] = true
		}
	}
	if len(included) > 2000 {
		return ledgerPage{}, fmt.Errorf("events: context exceeds 2000 events; reduce --max-matches or context")
	}
	keep := map[string]bool{}
	for _, s := range selection {
		keep[s] = true
	}
	selected := []ledgerEvent{}
	previous := -2
	for i, e := range events {
		if included[i] {
			out := ledgerEvent{}
			for k, v := range e {
				out[k] = v
			}
			for _, f := range []string{"message", "usage", "tools", "lifecycle"} {
				if !keep[f] {
					delete(out, f)
				}
			}
			if !payload {
				delete(out, "payload")
			}
			out["window_match"] = anchors[i]
			if i != previous+1 {
				wr.Windows++
				out["window_start"] = true
			}
			previous = i
			selected = append(selected, out)
		}
	}
	wr.ContextEvents = len(selected) - len(matches)
	receipt.Window = wr
	receipt.Selected = len(selected)
	if len(selected) > 0 {
		receipt.First = map[string]any{"event_id": selected[0]["event_id"], "ordinal": selected[0]["ordinal"]}
		receipt.Last = map[string]any{"event_id": selected[len(selected)-1]["event_id"], "ordinal": selected[len(selected)-1]["ordinal"]}
	}
	return buildLedgerPage(session, id, schema, detail, selection, payload, selected, page, supplied, "", all, receipt)
}
