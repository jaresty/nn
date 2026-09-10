package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type ledgerPage struct {
	All          bool                `json:"all,omitempty"`
	Query        *ledgerQueryReceipt `json:"query,omitempty"`
	Handoff      *handoffReceipt     `json:"handoff,omitempty"`
	Version      string              `json:"version"`
	EventFilter  string              `json:"event_filter"`
	Snapshot     string              `json:"snapshot"`
	Page         int                 `json:"page"`
	Pages        int                 `json:"pages"`
	NextPage     int                 `json:"next_page"`
	Select       []string            `json:"select"`
	Payload      bool                `json:"payload"`
	Schema       string              `json:"schema"`
	DetailStatus string              `json:"detail_status"`
	Events       []json.RawMessage   `json:"events"`
}

func ledgerSelect(s string) ([]string, error) {
	set := map[string]bool{"identity": true}
	for _, v := range strings.Split(s, ",") {
		switch v {
		case "identity", "message", "usage", "tools", "lifecycle":
			set[v] = true
		default:
			return nil, fmt.Errorf("events: unknown facet %q", v)
		}
	}
	out := []string{}
	for _, v := range []string{"identity", "message", "usage", "tools", "lifecycle"} {
		if set[v] {
			out = append(out, v)
		}
	}
	return out, nil
}

func newTranscriptEventsCmd() *cobra.Command {
	var selection, snapshot, eventFilter, summary, groupBy, since, until, at string
	var payload, asJSON, all, errorsOnly bool
	var page, bucketSize, resultLimit, last int
	var format string
	var maxTextChars int
	var window ledgerWindowOptions
	c := &cobra.Command{Use: "events <session> <agent-id>", Short: "Snapshot-bound event ledger (JSON) or bounded readable tail", Args: cobra.ExactArgs(2), RunE: func(c *cobra.Command, args []string) error {
		summaryMode := c.Flags().Changed("summary")
		if err := window.configure(c, eventFilter, format); err != nil {
			return err
		}
		if format != "json" && format != "text" {
			return fmt.Errorf("events: --format must be json or text")
		}
		if format == "text" {
			if (!window.Enabled && last < 1) || last < 0 || last > 200 || maxTextChars < 1 || maxTextChars > 10000 {
				return fmt.Errorf("events: text requires --last 1..200 and --max-text-chars 1..10000")
			}
			for _, flag := range []string{"json", "all", "page", "snapshot", "event", "at", "summary", "select", "payload"} {
				if flag == "event" && window.Enabled {
					continue
				}
				if c.Flags().Changed(flag) {
					return fmt.Errorf("events: text cannot combine with --%s", flag)
				}
			}
			selection = "identity,message,tools,lifecycle"
			payload = true
		} else if c.Flags().Changed("max-text-chars") {
			return fmt.Errorf("events: --max-text-chars requires --format text")
		}
		if c.Flags().Changed("last") {
			if last <= 0 {
				return fmt.Errorf("events: --last must be greater than zero")
			}
			for _, flag := range []string{"all", "event", "summary", "at"} {
				if c.Flags().Changed(flag) {
					return fmt.Errorf("events: --last cannot be combined with --%s", flag)
				}
			}
		}
		if c.Flags().Changed("at") {
			if at != "launch" && at != "return" {
				return fmt.Errorf("events: --at must be launch or return")
			}
			for _, flag := range []string{"summary", "since", "until", "errors-only"} {
				if c.Flags().Changed(flag) {
					return fmt.Errorf("events: --at cannot be combined with --%s", flag)
				}
			}
		}
		query := ledgerQuery{ErrorsOnly: errorsOnly, Last: last}
		for _, flag := range []string{"since", "until", "errors-only"} {
			if c.Flags().Changed(flag) && (summaryMode || c.Flags().Changed("event")) {
				return fmt.Errorf("events: --%s cannot be combined with --summary or --event", flag)
			}
		}
		var boundErr error
		if c.Flags().Changed("since") {
			query.Since, boundErr = parseLedgerBound(since)
			if boundErr != nil {
				return boundErr
			}
		}
		if c.Flags().Changed("until") {
			query.Until, boundErr = parseLedgerBound(until)
			if boundErr != nil {
				return boundErr
			}
		}
		if query.Since != nil && query.Until != nil && query.Since.After(*query.Until) {
			return fmt.Errorf("events: --since must not be later than --until")
		}
		if c.Flags().Changed("bucket-size") && summary != "usage" {
			return fmt.Errorf("events: --bucket-size requires --summary usage")
		}
		if c.Flags().Changed("group-by") && summary != "tools" {
			return fmt.Errorf("events: --group-by requires --summary tools")
		}
		if c.Flags().Changed("limit") && summary != "tools" && summary != "timing" {
			return fmt.Errorf("events: --limit requires --summary tools or timing")
		}
		if summaryMode {
			if summary != "usage" && summary != "tools" && summary != "timing" {
				return fmt.Errorf("events: unknown summary %q", summary)
			}
			if bucketSize < 0 {
				return fmt.Errorf("events: --bucket-size must not be negative")
			}
			for _, flag := range []string{"select", "payload", "event", "all", "page"} {
				if c.Flags().Changed(flag) {
					return fmt.Errorf("events: --summary cannot be combined with --%s", flag)
				}
			}
			selection = "identity,usage"
			if summary == "timing" {
				if resultLimit < 0 || resultLimit > 100 {
					return fmt.Errorf("events: timing summary requires --limit 0..100")
				}
				selection = "identity,message,tools"
			}
			if summary == "tools" {
				if resultLimit < 0 || resultLimit > 100 || (groupBy != "" && groupBy != "tool") {
					return fmt.Errorf("events: tools summary requires --limit 0..100 and --group-by tool")
				}
				selection = "identity,tools"
			}
		} else if c.Flags().Changed("bucket-size") {
			return fmt.Errorf("events: --bucket-size requires --summary usage")
		}
		if all && (c.Flags().Changed("page") || c.Flags().Changed("snapshot")) {
			return fmt.Errorf("events: --all cannot be combined with --page or --snapshot")
		}
		if args[1] == "" {
			return fmt.Errorf("events: agent id must not be empty")
		}
		if !asJSON {
			return fmt.Errorf("events: only JSON output is supported")
		}
		if page < 1 {
			return fmt.Errorf("events: --page must be at least 1")
		}
		if page > 1 && snapshot == "" {
			return fmt.Errorf("events: --snapshot required for later pages")
		}
		selectFields, err := ledgerSelect(selection)
		if err != nil {
			return err
		}
		if at != "" {
			result, err := buildHandoffPage(args[0], args[1], at, selectFields, payload, page, snapshot, eventFilter, all)
			if err != nil {
				return err
			}
			b, err := json.Marshal(result)
			if err != nil {
				return err
			}
			_, err = c.OutOrStdout().Write(append(b, '\n'))
			return err
		}
		records, schema, detail, err := ledgerRecords(args[0], args[1])
		if err != nil {
			return err
		}
		projectionFields := selectFields
		if errorsOnly {
			projectionFields, _ = ledgerSelect(selection + ",message,tools")
		}
		if window.Enabled {
			projectionFields, _ = ledgerSelect("identity,message,usage,tools,lifecycle")
		}
		events, err := projectLedger(records, args[1], projectionFields, payload || summary == "tools" || window.Enabled)
		if err != nil {
			return err
		}
		if summaryMode {
			var body []byte
			if summary == "timing" {
				body, err = buildTimingSummary(args[0], args[1], schema, detail, events, resultLimit, snapshot)
			} else if summary == "tools" {
				body, err = buildToolSummary(args[0], args[1], schema, detail, events, resultLimit, groupBy, snapshot)
			} else {
				body, err = buildUsageSummary(args[0], args[1], schema, detail, events, bucketSize, snapshot)
			}
			if err != nil {
				return err
			}
			_, err = c.OutOrStdout().Write(body)
			return err
		}
		var result ledgerPage
		if window.Enabled {
			result, err = buildWindowLedgerPage(args[0], args[1], schema, detail, selectFields, payload, events, page, snapshot, eventFilter, all || format == "text", query, window)
		} else {
			result, err = buildQueriedLedgerPage(args[0], args[1], schema, detail, selectFields, payload, events, page, snapshot, eventFilter, all || format == "text", query)
		}
		if err != nil {
			return err
		}
		if format == "text" {
			if window.Enabled {
				var output bytes.Buffer
				if err := renderLedgerText(&output, result, maxTextChars); err != nil {
					return err
				}
				if output.Len() > 200000 {
					return fmt.Errorf("events: text exceeds 200000 bytes; reduce --max-matches, context, or --max-text-chars")
				}
				_, err = c.OutOrStdout().Write(output.Bytes())
				return err
			}
			return renderLedgerText(c.OutOrStdout(), result, maxTextChars)
		}
		b, err := json.Marshal(result)
		if err != nil {
			return err
		}
		_, err = c.OutOrStdout().Write(append(b, '\n'))
		return err
	}}
	window.flags(c)
	c.Flags().StringVar(&format, "format", "json", "json or bounded readable text (requires --last, --event, or search/context options)")
	c.Flags().IntVar(&maxTextChars, "max-text-chars", 1000, "per-event readable character limit, 1..10000 (text only)")
	c.Flags().StringVar(&at, "at", "", "parent-side handoff occurrences: launch or return (Pi)")
	c.Flags().StringVar(&since, "since", "", "inclusive RFC3339 lower event timestamp bound")
	c.Flags().StringVar(&until, "until", "", "inclusive RFC3339 upper event timestamp bound")
	c.Flags().BoolVar(&errorsOnly, "errors-only", false, "select recorded assistant failures and explicitly erroneous tool results")
	c.Flags().IntVar(&last, "last", 0, "return the latest N matching events in canonical ledger order")
	c.Flags().StringVar(&summary, "summary", "", "deterministic summary: usage, tools, or timing")
	c.Flags().IntVar(&resultLimit, "limit", 5, "largest tool results or timing intervals to return, 0..100 (requires --summary tools or timing)")
	c.Flags().StringVar(&groupBy, "group-by", "", "group tool-volume statistics by tool (requires --summary tools)")
	c.Flags().IntVar(&bucketSize, "bucket-size", 0, "usage records per summary bucket; 0 disables buckets (requires --summary usage)")
	c.Flags().BoolVar(&all, "all", false, "export all complete events as UNBOUNDED JSON; incompatible with paging flags")
	c.Flags().StringVar(&eventFilter, "event", "", "retrieve one exact event id, preserving its ledger ordinal")
	c.Flags().StringVar(&selection, "select", "identity,message,usage,tools,lifecycle", "comma-separated facets; identity is always included")
	c.Flags().BoolVar(&payload, "payload", false, "include native payloads (oversized events are fragmented)")
	c.Flags().BoolVar(&asJSON, "json", true, "emit bounded JSON (default; use --format text for readable tails)")
	c.Flags().IntVar(&page, "page", 1, "one-based page")
	c.Flags().StringVar(&snapshot, "snapshot", "", "page-1 SHA-256; required for every later page")
	return c
}

func buildLedgerPage(session, id, schema, detail string, selection []string, payload bool, events []ledgerEvent, page int, supplied, eventFilter string, all bool, queries ...*ledgerQueryReceipt) (ledgerPage, error) {
	return buildLedgerPageWithHandoff(session, id, schema, detail, selection, payload, events, page, supplied, eventFilter, all, nil, queries...)
}

func buildLedgerPageWithHandoff(session, id, schema, detail string, selection []string, payload bool, events []ledgerEvent, page int, supplied, eventFilter string, all bool, handoff *handoffReceipt, queries ...*ledgerQueryReceipt) (ledgerPage, error) {
	if page < 1 || (page > 1 && supplied == "") {
		return ledgerPage{}, fmt.Errorf("events: invalid page or missing snapshot")
	}
	if eventFilter != "" {
		selected := []ledgerEvent{}
		for _, e := range events {
			if e["event_id"] == eventFilter {
				selected = append(selected, e)
			}
		}
		if len(selected) != 1 {
			return ledgerPage{}, fmt.Errorf("events: event id not found")
		}
		events = selected
	}
	absolute, err := filepath.Abs(session)
	if err != nil {
		return ledgerPage{}, err
	}
	request, _ := json.Marshal([]any{filepath.Clean(absolute), id, selection, payload})
	result := ledgerPage{Version: "nn.transcript.events/v1", EventFilter: eventFilter, Select: selection, Payload: payload, Schema: schema, DetailStatus: detail, Events: []json.RawMessage{}}
	if len(queries) > 0 {
		result.Query = queries[0]
	}
	result.Handoff = handoff
	header, _ := json.Marshal(result)
	h := sha256.New()
	writeSnapshotPart(h, []byte("nn transcript events snapshot v1"))
	writeSnapshotPart(h, request)
	writeSnapshotPart(h, header)
	encoded := make([]json.RawMessage, len(events))
	for i, e := range events {
		b, err := json.Marshal(e)
		if err != nil {
			return ledgerPage{}, err
		}
		encoded[i] = b
		writeSnapshotPart(h, b)
	}
	result.Snapshot = hex.EncodeToString(h.Sum(nil))
	if supplied != "" && supplied != result.Snapshot {
		return ledgerPage{}, fmt.Errorf("events: stale or mismatched --snapshot")
	}
	if all {
		return completeLedgerExport(result, encoded), nil
	}
	fits := func(entries []json.RawMessage) bool {
		p := result
		p.Events = entries
		p.Page = math.MaxInt64
		p.Pages = math.MaxInt64
		p.NextPage = math.MaxInt64
		b, err := json.Marshal(p)
		return err == nil && len(b)+1 <= graphBodiesPageMaxBytes
	}
	entries := []json.RawMessage{}
	for i, b := range encoded {
		if fits([]json.RawMessage{b}) {
			entries = append(entries, b)
			continue
		}
		chunks := splitGraphBody(string(b))
		for j, text := range chunks {
			fragment, _ := json.Marshal(map[string]any{"event_id": events[i]["event_id"], "ordinal": events[i]["ordinal"], "segment": j + 1, "segments": len(chunks), "text": text})
			if !fits([]json.RawMessage{fragment}) {
				return ledgerPage{}, fmt.Errorf("events: fragment cannot fit page")
			}
			entries = append(entries, fragment)
		}
	}
	pages := [][]json.RawMessage{{}}
	for _, entry := range entries {
		last := len(pages) - 1
		candidate := append(append([]json.RawMessage{}, pages[last]...), entry)
		if fits(candidate) {
			pages[last] = candidate
		} else {
			pages = append(pages, []json.RawMessage{entry})
		}
	}
	if page > len(pages) {
		return ledgerPage{}, fmt.Errorf("events: page out of range")
	}
	result.Page = page
	result.Pages = len(pages)
	result.Events = pages[page-1]
	if result.Events == nil {
		result.Events = []json.RawMessage{}
	}
	if page < len(pages) {
		result.NextPage = page + 1
	}
	b, err := json.Marshal(result)
	if err != nil {
		return ledgerPage{}, err
	}
	if len(b)+1 > graphBodiesPageMaxBytes {
		return ledgerPage{}, fmt.Errorf("events: encoded page exceeds limit")
	}
	return result, nil
}
