package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Both independently bounded selections consume the same acquired projection.
func renderCombinedEventTails(w io.Writer, session, id, schema, detail string, fields []string, events []ledgerEvent, q ledgerQuery, failures, maxChars int) error {
	recent, err := buildQueriedLedgerPage(session, id, schema, detail, fields, true, events, 1, "", "", true, q)
	if err != nil {
		return err
	}
	q.ErrorsOnly = true
	q.Last = failures
	errors, err := buildQueriedLedgerPage(session, id, schema, detail, fields, true, events, 1, "", "", true, q)
	if err != nil {
		return err
	}
	binding, err := json.Marshal([]ledgerPage{recent, errors})
	if err != nil {
		return err
	}
	snapshot := captureHash(binding)
	recent.Snapshot = snapshot
	errors.Snapshot = snapshot
	ids := map[string]bool{}
	for _, raw := range recent.Events {
		var e map[string]json.RawMessage
		if err := json.Unmarshal(raw, &e); err != nil {
			return err
		}
		ids[ledgerString(e, "event_id")] = true
	}
	overlap := []string{}
	for _, raw := range errors.Events {
		var e map[string]json.RawMessage
		if err := json.Unmarshal(raw, &e); err != nil {
			return err
		}
		id := ledgerString(e, "event_id")
		if ids[id] {
			overlap = append(overlap, id)
		}
	}
	var out bytes.Buffer
	fmt.Fprintf(&out, "Combined event tails · one ledger acquisition\nShared snapshot: %s\nOverlapping event IDs (%d): %v\n\n## Latest events\n", snapshot, len(overlap), overlap)
	if err := renderLedgerText(&out, recent, maxChars); err != nil {
		return err
	}
	fmt.Fprintln(&out, "\n## Latest explicit failures")
	if err := renderLedgerText(&out, errors, maxChars); err != nil {
		return err
	}
	fmt.Fprintln(&out, "Failure events may predate the recent tail; their presence does not establish that they remain unresolved. No failures is not a health verdict. Snapshot identifies this combined view, not retained replay or an atomic multi-source capture.")
	if out.Len() > 200000 {
		return fmt.Errorf("events: combined text exceeds 200000 bytes; reduce tail limits or --max-text-chars")
	}
	_, err = w.Write(out.Bytes())
	return err
}
