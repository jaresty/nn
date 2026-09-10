package cmd

import (
	"fmt"
	"strings"
	"time"
)

type ledgerQuery struct {
	Since      *time.Time
	Until      *time.Time
	ErrorsOnly bool
	Last       int
}

func (q ledgerQuery) active() bool {
	return q.Since != nil || q.Until != nil || q.ErrorsOnly || q.Last > 0
}

type ledgerQueryReceipt struct {
	Window         *ledgerWindowReceipt `json:"window,omitempty"`
	Since          *string              `json:"since"`
	Until          *string              `json:"until"`
	ErrorsOnly     bool                 `json:"errors_only"`
	Clock          string               `json:"clock"`
	Boundary       string               `json:"boundary"`
	LedgerSnapshot string               `json:"ledger_snapshot"`
	Total          int                  `json:"total_events"`
	Selected       int                  `json:"selected_events"`
	Before         int                  `json:"excluded_before"`
	After          int                  `json:"excluded_after"`
	Unknown        int                  `json:"excluded_unknown_timestamp"`
	NonErrors      int                  `json:"excluded_non_errors"`
	First          any                  `json:"first_event"`
	Last           any                  `json:"last_event"`
	Completeness   string               `json:"completeness"`
	RequestedLast  *int                 `json:"requested_last,omitempty"`
	MatchingEvents *int                 `json:"matching_events,omitempty"`
	ReturnedEvents *int                 `json:"returned_events,omitempty"`
	OlderMatching  *bool                `json:"older_matching_events,omitempty"`
}

func parseLedgerBound(value string) (*time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, fmt.Errorf("events: window bounds must be nonempty RFC3339 timestamps: %q", value)
	}
	return &t, nil
}

func buildQueriedLedgerPage(session, id, schema, detail string, selection []string, payload bool, events []ledgerEvent, page int, supplied, eventFilter string, all bool, q ledgerQuery) (ledgerPage, error) {
	if !q.active() {
		return buildLedgerPage(session, id, schema, detail, selection, payload, events, page, supplied, eventFilter, all)
	}
	if eventFilter != "" {
		return ledgerPage{}, fmt.Errorf("events: --event cannot be combined with window or error filters")
	}
	if q.Since != nil && q.Until != nil && q.Since.After(*q.Until) {
		return ledgerPage{}, fmt.Errorf("events: --since must not be later than --until")
	}
	// Error classification must be performed before caller facet suppression.
	// Bind the whole evidence projection, including out-of-window records, so
	// continuation cannot silently combine two different retained histories.
	fields := append([]string{}, selection...)
	if q.ErrorsOnly {
		fields = append(fields, "message", "tools")
	}
	canonical, err := ledgerSelect(strings.Join(fields, ","))
	if err != nil {
		return ledgerPage{}, err
	}
	full, err := buildLedgerPage(session, id, schema, detail, canonical, payload, events, 1, "", "", true)
	if err != nil {
		return ledgerPage{}, err
	}
	receipt := &ledgerQueryReceipt{ErrorsOnly: q.ErrorsOnly, Clock: "record_preferred_message_fallback", Boundary: "inclusive", LedgerSnapshot: full.Snapshot, Total: len(events), Completeness: "Complete selected projection only after every page/segment is retrieved; not original-source completeness."}
	if q.Since != nil {
		s := q.Since.UTC().Format(time.RFC3339Nano)
		receipt.Since = &s
	}
	if q.Until != nil {
		s := q.Until.UTC().Format(time.RFC3339Nano)
		receipt.Until = &s
	}
	selected := []ledgerEvent{}
	keep := map[string]bool{}
	for _, s := range selection {
		keep[s] = true
	}
	for _, e := range events {
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
		// Copy to leave the full authenticated projection and its joins untouched.
		out := ledgerEvent{}
		for k, v := range e {
			out[k] = v
		}
		for _, facet := range []string{"message", "usage", "tools", "lifecycle"} {
			if !keep[facet] {
				delete(out, facet)
			}
		}
		selected = append(selected, out)
	}
	if q.Last > 0 {
		matching := len(selected)
		start := matching - q.Last
		if start < 0 {
			start = 0
		}
		selected = selected[start:]
		returned := len(selected)
		older := matching > returned
		receipt.RequestedLast = &q.Last
		receipt.MatchingEvents = &matching
		receipt.ReturnedEvents = &returned
		receipt.OlderMatching = &older
	}
	receipt.Selected = len(selected)
	endpoint := func(e ledgerEvent) any {
		return map[string]any{"event_id": e["event_id"], "ordinal": e["ordinal"], "timestamp": e["timestamp"]}
	}
	if len(selected) > 0 {
		receipt.First = endpoint(selected[0])
		receipt.Last = endpoint(selected[len(selected)-1])
	}
	return buildLedgerPage(session, id, schema, detail, selection, payload, selected, page, supplied, "", all, receipt)
}
