package cmd

import (
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
	All          bool              `json:"all,omitempty"`
	Version      string            `json:"version"`
	EventFilter  string            `json:"event_filter"`
	Snapshot     string            `json:"snapshot"`
	Page         int               `json:"page"`
	Pages        int               `json:"pages"`
	NextPage     int               `json:"next_page"`
	Select       []string          `json:"select"`
	Payload      bool              `json:"payload"`
	Schema       string            `json:"schema"`
	DetailStatus string            `json:"detail_status"`
	Events       []json.RawMessage `json:"events"`
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
	var selection, snapshot, eventFilter string
	var payload, asJSON, all bool
	var page int
	c := &cobra.Command{Use: "events <session> <agent-id>", Short: "Snapshot-bound normalized event ledger (JSON)", Args: cobra.ExactArgs(2), RunE: func(c *cobra.Command, args []string) error {
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
		records, schema, detail, err := ledgerRecords(args[0], args[1])
		if err != nil {
			return err
		}
		events, err := projectLedger(records, args[1], selectFields, payload)
		if err != nil {
			return err
		}
		result, err := buildLedgerPage(args[0], args[1], schema, detail, selectFields, payload, events, page, snapshot, eventFilter, all)
		if err != nil {
			return err
		}
		b, err := json.Marshal(result)
		if err != nil {
			return err
		}
		_, err = c.OutOrStdout().Write(append(b, '\n'))
		return err
	}}
	c.Flags().BoolVar(&all, "all", false, "export all complete events as UNBOUNDED JSON; incompatible with paging flags")
	c.Flags().StringVar(&eventFilter, "event", "", "retrieve one exact event id, preserving its ledger ordinal")
	c.Flags().StringVar(&selection, "select", "identity,message,usage,tools,lifecycle", "comma-separated facets; identity is always included")
	c.Flags().BoolVar(&payload, "payload", false, "include native payloads (oversized events are fragmented)")
	c.Flags().BoolVar(&asJSON, "json", true, "emit bounded JSON (the only output format)")
	c.Flags().IntVar(&page, "page", 1, "one-based page")
	c.Flags().StringVar(&snapshot, "snapshot", "", "page-1 SHA-256; required for every later page")
	return c
}

func buildLedgerPage(session, id, schema, detail string, selection []string, payload bool, events []ledgerEvent, page int, supplied, eventFilter string, all bool) (ledgerPage, error) {
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
