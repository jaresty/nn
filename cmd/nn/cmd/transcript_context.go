package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Bundles reuse ledger transport, including lossless oversized-record segments.
// Their first record is a bundle receipt; each room has its own metadata record.
// Source event identities are never replaced with bundle-relative identities.
func newTranscriptContextCmd() *cobra.Command {
	var last, page int
	var snapshot string
	var asJSON bool
	var text bundleTextOptions
	c := &cobra.Command{Use: "context <session> <agent-id>", Short: "Recorded launch assignments and bounded recent evidence; no inferred governing attempt", Args: cobra.ExactArgs(2), RunE: func(c *cobra.Command, args []string) error {
		if err := text.validate(c); err != nil {
			return err
		}
		if !asJSON && text.Format != "text" {
			return fmt.Errorf("context: --json is required")
		}
		result, err := buildTranscriptContext(args[0], args[1], last, page, snapshot)
		if err != nil {
			return err
		}
		if text.Format == "text" {
			return renderBundleText(c.OutOrStdout(), result, text, func(n int) (ledgerPage, error) {
				return buildTranscriptContext(args[0], args[1], last, n, result.Snapshot)
			})
		}
		return json.NewEncoder(c.OutOrStdout()).Encode(result)
	}}
	addBundleTextFlags(c, &text)
	c.Flags().BoolVar(&asJSON, "json", false, "emit lossless paginated context JSON")
	c.Flags().IntVar(&last, "last", 5, "recent events per room, 1–200")
	c.Flags().IntVar(&page, "page", 1, "transport page, not a new room selection")
	c.Flags().StringVar(&snapshot, "snapshot", "", "bundle snapshot required for later transport pages")
	return c
}

func contextPath(session string) (string, error) {
	path, e := filepath.Abs(session)
	if e != nil {
		return "", e
	}
	return filepath.EvalSymlinks(path)
}

func contextBounds(last, page int, snapshot string) error {
	if last < 1 || last > 200 {
		return fmt.Errorf("context: --last must be 1..200")
	}
	if page < 1 || (page > 1 && snapshot == "") {
		return fmt.Errorf("context: positive --page and --snapshot for later pages required")
	}
	return nil
}

func buildTranscriptContext(session, id string, last, page int, snapshot string) (ledgerPage, error) {
	var empty ledgerPage
	if e := contextBounds(last, page, snapshot); e != nil {
		return empty, e
	}
	request, e := captureRequest(session, "context", id, last)
	if e != nil {
		return empty, e
	}
	if snapshot != "" {
		return loadCapturedPage(snapshot, request, page)
	}
	capture, e := newTranscriptCapture(session)
	if e != nil {
		return empty, e
	}
	path := capture.Path
	agents, e := buildPiTreeUsing(path, capture.read, capture.resolve)
	if e != nil {
		return empty, e
	}
	known := false
	for _, a := range agents {
		if a.ID == id {
			known = true
			break
		}
	}
	if !known || id == "" {
		return empty, fmt.Errorf("context: unknown agent %q", id)
	}
	events := []ledgerEvent{}
	fields, _ := ledgerSelect("identity,message,usage,tools,lifecycle")
	launches, e := buildHandoffPage(path, id, "launch", fields, true, 1, "", "", true, capture)
	if e != nil {
		return empty, e
	}
	tail, sources, e := contextTail(capture, id, last, true)
	if e != nil {
		return empty, e
	}
	// A source fingerprint is included even when a changed source produces the same tail.

	receipt := ledgerEvent{"event_id": ledgerID(path, 0, id, "context:receipt"), "kind": "context_receipt", "ordinal": 0, "agent_id": id, "bundle": "assignment_context", "path": path, "last": last, "launches": launches.Handoff, "recent": tail.Query, "steering_status": "unavailable", "governing_attempt": "not_inferred", "detail_status": tail.DetailStatus, "capture_sources": len(sources), "capture_id": capture.ID, "capture_boundary": "sequential_complete_record_prefixes"}
	events = append(events, receipt)
	events, e = appendContextEvents(events, launches.Events, "launch")
	if e != nil {
		return empty, e
	}
	events, e = appendContextEvents(events, tail.Events, "recent")
	if e != nil {
		return empty, e
	}
	result, e := buildLedgerPage(path, id, schemaPi, tail.DetailStatus, fields, true, events, page, snapshot, "", false)
	if e != nil {
		return empty, e
	}
	if e = saveCapturedPages(result, request, capture, func(n int) (ledgerPage, error) {
		return buildLedgerPage(path, id, schemaPi, tail.DetailStatus, fields, true, events, n, result.Snapshot, "", false)
	}); e != nil {
		return empty, e
	}
	return result, nil
}

func buildReviewTails(session, queue, order, pattern string, limit int, cursor string, last int, payload bool, page int, snapshot string) (ledgerPage, error) {
	var empty ledgerPage
	if e := contextBounds(last, page, snapshot); e != nil {
		return empty, e
	}
	request, e := captureRequest(session, "review", queue, order, pattern, limit, cursor, last, payload)
	if e != nil {
		return empty, e
	}
	if snapshot != "" {
		return loadCapturedPage(snapshot, request, page)
	}
	capture, e := reviewCapture(session, cursor)
	if e != nil {
		return empty, e
	}
	path := capture.Path
	rooms, e := buildReviewPageCaptured(capture, queue, order, pattern, limit, cursor)
	if e != nil {
		return empty, e
	}
	fields, _ := ledgerSelect("identity,message,usage,tools,lifecycle")
	events := []ledgerEvent{}
	sources := capture.digests()
	retrieved, unavailable, omittedEvents := 0, 0, 0
	for i, row := range rooms.Rows {
		tail, digests, e := contextTail(capture, row.ID, last, payload)
		if e != nil {
			return empty, e
		}
		for source, digest := range digests {
			if previous, ok := sources[source]; ok && previous != digest {
				return empty, fmt.Errorf("review: source changed during bundle collection")
			}
			sources[source] = digest
		}
		if tail.DetailStatus == "unavailable" {
			unavailable++
		}
		retrieved += len(tail.Events)
		omittedEvents += tail.Query.Total - tail.Query.Selected
		events = append(events, ledgerEvent{"event_id": ledgerID(path, i, row.ID, "review:room"), "ordinal": i, "kind": "review_room", "agent_id": row.ID, "room": row, "recent": tail.Query, "room_snapshot": tail.Snapshot, "detail_status": tail.DetailStatus})
		events, e = appendContextEvents(events, tail.Events, "recent")
		if e != nil {
			return empty, e
		}
	}
	receipt := ledgerEvent{"event_id": ledgerID(path, 0, "ROOT", "review:receipt"), "ordinal": 0, "kind": "review_receipt", "bundle": "review_tails", "path": path, "queue": queue, "order": order, "pattern": pattern, "last": last, "limit": limit, "review_snapshot": rooms.Snapshot, "population": rooms.Population, "eligible": rooms.Eligible, "offset": rooms.Offset, "retrieved_rooms": rooms.Returned, "omitted_rooms": rooms.Omitted, "unknown_population": rooms.Unknown, "unavailable_rooms": unavailable, "selected_events": retrieved, "omitted_earlier_events": omittedEvents, "next_room_cursor": rooms.NextCursor, "inspection_status": "not_inferred", "capture_sources": len(sources), "capture_id": capture.ID, "capture_boundary": "sequential_complete_record_prefixes"}
	events = append([]ledgerEvent{receipt}, events...)
	result, e := buildLedgerPage(path, "review_tails", schemaPi, "per_room", fields, payload, events, page, snapshot, "", false)
	if e != nil {
		return empty, e
	}
	if e = saveCapturedPages(result, request, capture, func(n int) (ledgerPage, error) {
		return buildLedgerPage(path, "review_tails", schemaPi, "per_room", fields, payload, events, n, result.Snapshot, "", false)
	}); e != nil {
		return empty, e
	}
	return result, nil
}

func appendContextEvents(events []ledgerEvent, raws []json.RawMessage, section string) ([]ledgerEvent, error) {
	for _, raw := range raws {
		var event ledgerEvent
		if e := json.Unmarshal(raw, &event); e != nil {
			return nil, e
		}
		event["section"] = section
		events = append(events, event)
	}
	return events, nil
}

func contextTail(capture *transcriptCapture, id string, last int, payload bool) (ledgerPage, map[string]string, error) {
	records, detail := capture.ledger(id)
	fields, _ := ledgerSelect("identity,message,usage,tools,lifecycle")
	events, e := projectLedger(records, id, fields, payload)
	if e != nil {
		return ledgerPage{}, nil, e
	}
	result, e := buildQueriedLedgerPage(capture.Path, id, schemaPi, detail, fields, payload, events, 1, "", "", true, ledgerQuery{Last: last})
	return result, capture.digests(), e
}
