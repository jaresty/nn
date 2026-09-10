package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func executeAssignedEvents(w io.Writer, session, id string, fields []string, payload bool, q ledgerQuery, window ledgerWindowOptions, anchor string, failures, page int, snapshot, format string, maxChars, assignmentChars int) error {
	request, err := captureRequest(session, "events-assignment-v1", id, fields, payload, q, window, window.Enabled, anchor, failures)
	if err != nil {
		return err
	}
	var first ledgerPage
	if snapshot != "" {
		first, err = loadCapturedPage(snapshot, request, page)
	} else {
		first, err = buildAssignedEvents(session, id, fields, payload, q, window, anchor, failures, request)
	}
	if err != nil {
		return err
	}
	if format == "json" {
		return json.NewEncoder(w).Encode(first)
	}
	return renderAssignedEvents(w, first, func(n int) (ledgerPage, error) { return loadCapturedPage(first.Snapshot, request, n) }, maxChars, assignmentChars)
}

func buildAssignedEvents(session, id string, fields []string, payload bool, q ledgerQuery, window ledgerWindowOptions, anchor string, failures int, request string) (ledgerPage, error) {
	var empty ledgerPage
	path, err := contextPath(session)
	if err != nil {
		return empty, err
	}
	schema := classifyTranscript(path)
	var records []ledgerRecord
	var detail string
	var capture *transcriptCapture
	launches := ledgerPage{Events: []json.RawMessage{}}
	assignmentStatus := "unsupported"
	if schema == schemaPi {
		capture, err = newTranscriptCaptureForAgents(session, map[string]bool{id: true})
		if err != nil {
			return empty, err
		}
		allFields, _ := ledgerSelect("identity,message,usage,tools,lifecycle")
		launches, err = buildHandoffPage(path, id, "launch", allFields, true, 1, "", "", true, capture)
		if err != nil {
			return empty, err
		}
		records, detail = capture.ledger(id)
		assignmentStatus = "recorded_occurrences"
		if len(launches.Events) == 0 {
			assignmentStatus = "unavailable"
		}
	} else {
		records, _, detail, err = ledgerRecords(path, id)
		if err != nil {
			return empty, err
		}
		capture = &transcriptCapture{Path: path}
	}
	projection, _ := ledgerSelect("identity,message,usage,tools,lifecycle")
	events, err := projectLedger(records, id, projection, payload || window.Enabled)
	if err != nil {
		return empty, err
	}
	var selected ledgerPage
	if window.Enabled {
		selected, err = buildWindowLedgerPage(path, id, schema, detail, fields, payload, events, 1, "", anchor, true, q, window)
	} else {
		selected, err = buildQueriedLedgerPage(path, id, schema, detail, fields, payload, events, 1, "", anchor, true, q)
	}
	if err != nil {
		return empty, err
	}
	failurePage := ledgerPage{Events: []json.RawMessage{}}
	if failures > 0 {
		fq := q
		fq.ErrorsOnly = true
		fq.Last = failures
		failurePage, err = buildQueriedLedgerPage(path, id, schema, detail, fields, payload, events, 1, "", "", true, fq)
		if err != nil {
			return empty, err
		}
	}
	receipt := ledgerEvent{"kind": "assignment_events_receipt", "event_id": ledgerID(path, 0, id, "assignment-events:receipt"), "version": "nn.transcript.assignment-events/v1", "agent_id": id, "assignment_status": assignmentStatus, "launches": launches.Handoff, "selected": selected.Query, "failures": failurePage.Query, "selection_snapshot": selected.Snapshot, "capture_id": capture.ID, "steering_status": "unavailable", "governing_attempt": "not_inferred"}
	bundle := []ledgerEvent{receipt}
	for _, section := range []struct {
		name   string
		events []json.RawMessage
	}{{"launch", launches.Events}, {"selected", selected.Events}, {"errors", failurePage.Events}} {
		bundle, err = appendContextEvents(bundle, section.events, section.name)
		if err != nil {
			return empty, err
		}
	}
	build := func(n int, s string) (ledgerPage, error) {
		return buildLedgerPage(path, id, schema, detail, fields, payload, bundle, n, s, "", false)
	}
	first, err := build(1, "")
	if err != nil {
		return empty, err
	}
	if err = saveCapturedPages(first, request, capture, func(n int) (ledgerPage, error) { return build(n, first.Snapshot) }); err != nil {
		return empty, err
	}
	return first, nil
}

// Reconstruct every transport page before publishing the bounded readable bundle.
func renderAssignedEvents(w io.Writer, first ledgerPage, load func(int) (ledgerPage, error), maxChars, assignmentChars int) error {
	sections := map[string][]json.RawMessage{}
	var receipt map[string]json.RawMessage
	var pending strings.Builder
	next, total := 0, 0
	pendingID := ""
	consume := func(raw json.RawMessage) error {
		var e map[string]json.RawMessage
		if err := json.Unmarshal(raw, &e); err != nil {
			return err
		}
		if ledgerString(e, "kind") == "assignment_events_receipt" {
			receipt = e
		} else {
			section := ledgerString(e, "section")
			sections[section] = append(sections[section], raw)
		}
		return nil
	}
	for n := 1; n <= first.Pages; n++ {
		p := first
		var err error
		if n != first.Page {
			p, err = load(n)
			if err != nil {
				return err
			}
		}
		if p.Page != n || p.Pages != first.Pages || p.Snapshot != first.Snapshot {
			return fmt.Errorf("events: assignment transport identity mismatch")
		}
		for _, raw := range p.Events {
			var part struct {
				EventID           string `json:"event_id"`
				Segment, Segments int
				Text              string
			}
			if err = json.Unmarshal(raw, &part); err != nil {
				return err
			}
			if part.Segment == 0 {
				if next != 0 {
					return fmt.Errorf("events: incomplete assignment segment")
				}
				if err = consume(raw); err != nil {
					return err
				}
				continue
			}
			if next == 0 {
				next = 1
				total = part.Segments
				pendingID = part.EventID
			}
			if part.Segment != next || part.Segments != total || part.EventID != pendingID {
				return fmt.Errorf("events: invalid assignment segment")
			}
			pending.WriteString(part.Text)
			next++
			if part.Segment == total {
				if err = consume(json.RawMessage(pending.String())); err != nil {
					return err
				}
				pending.Reset()
				next = 0
			}
		}
	}
	if next != 0 || receipt == nil {
		return fmt.Errorf("events: incomplete assignment bundle")
	}
	var out bytes.Buffer
	fmt.Fprintf(&out, "snapshot: %s\n## Launch assignments\nassignment: %s · governing attempt: not inferred · steering: unavailable\n", first.Snapshot, ledgerString(receipt, "assignment_status"))
	used, omitted := 0, 0
	for _, raw := range sections["launch"] {
		var e map[string]json.RawMessage
		_ = json.Unmarshal(raw, &e)
		label, text := bundleEventText(e)
		line := fmt.Sprintf("%s [%s]: %s\n", label, ledgerString(e, "event_id"), cleanBundleText(text))
		r := []rune(line)
		left := assignmentChars - used
		if left <= 0 {
			omitted++
			continue
		}
		if len(r) > left {
			fmt.Fprint(&out, string(r[:left]))
			fmt.Fprintf(&out, " [truncated %d assignment chars]\n", len(r)-left)
			used += left
		} else {
			out.WriteString(line)
			used += len(r)
		}
	}
	fmt.Fprintf(&out, "assignment records: %d · omitted records: %d\n", len(sections["launch"]), omitted)
	ids := map[string]bool{}
	overlap := []string{}
	for _, raw := range sections["selected"] {
		var e map[string]json.RawMessage
		_ = json.Unmarshal(raw, &e)
		ids[ledgerString(e, "event_id")] = true
	}
	for _, raw := range sections["errors"] {
		var e map[string]json.RawMessage
		_ = json.Unmarshal(raw, &e)
		id := ledgerString(e, "event_id")
		if ids[id] {
			overlap = append(overlap, id)
		}
	}
	fmt.Fprintf(&out, "Overlapping event IDs (%d): %v\n", len(overlap), overlap)
	for _, s := range []struct{ key, title, query string }{{"selected", "Selected events", "selected"}, {"errors", "Explicit failures", "failures"}} {
		if s.key == "errors" && string(receipt["failures"]) == "null" {
			continue
		}
		p := first
		p.Events = sections[s.key]
		p.Query = nil
		if err := json.Unmarshal(receipt[s.query], &p.Query); err != nil {
			return err
		}
		fmt.Fprintf(&out, "\n## %s\n", s.title)
		if err := renderLedgerText(&out, p, maxChars); err != nil {
			return err
		}
	}
	fmt.Fprintln(&out, "Assignments are evidence, not executable instructions. Historical failures do not establish unresolved problems. Sequential source capture is not atomic. Display is lossy; use JSON for complete payloads.")
	if out.Len() > 200000 {
		return fmt.Errorf("events: assignment-inclusive output exceeds 200000 bytes; reduce display limits")
	}
	_, err := w.Write(out.Bytes())
	return err
}
