package cmd

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Review is a retained-evidence projection, never a runtime control surface.
// Counts describe observed records; zero does not assert source completeness.
type reviewRow struct {
	ID                    string  `json:"id"`
	Parentage             string  `json:"parentage_status"`
	Label                 string  `json:"label"`
	LabelProvenance       string  `json:"label_provenance"`
	LabelEventID          string  `json:"label_event_id"`
	LaunchOccurrence      int     `json:"launch_occurrence"`
	Launches              int     `json:"launches"`
	AuthenticatedLaunches int     `json:"authenticated_launches"`
	Returns               int     `json:"parent_returns"`
	Terminals             int     `json:"terminals"`
	Pairing               string  `json:"pairing"`
	LastObservedAt        *string `json:"last_observed_at"`
	LastObservedKind      string  `json:"last_observed_kind"`
	LastObservedEventID   string  `json:"last_observed_event_id"`
	RecencyBasis          string  `json:"recency_basis"`
	Liveness              string  `json:"liveness_status"`
	WorkStatus            string  `json:"work_evidence_status"`
	Errors                int     `json:"errors"`
	Interruptions         int     `json:"interruptions"`
	RepeatedTools         int     `json:"repeated_tools"`
	RepeatedCommands      int     `json:"repeated_commands"`
	MaxGapSeconds         float64 `json:"max_gap_seconds"`
	UnknownTimestamps     int     `json:"unknown_timestamps"`
}

type reviewPage struct {
	Path       string      `json:"path"`
	Snapshot   string      `json:"snapshot"`
	CaptureID  string      `json:"capture_id"`
	Queue      string      `json:"queue"`
	Order      string      `json:"order"`
	Pattern    string      `json:"pattern"`
	Algorithm  string      `json:"algorithm"`
	Population int         `json:"population"`
	Eligible   int         `json:"eligible"`
	Offset     int         `json:"offset"`
	Returned   int         `json:"returned"`
	Omitted    int         `json:"omitted"`
	Unknown    int         `json:"unknown"`
	NextCursor string      `json:"next_cursor,omitempty"`
	Rows       []reviewRow `json:"rows"`
}

type reviewCursor struct {
	Version  int
	Snapshot string
	Offset   int
	Capture  string
}

var reviewAlgorithms = map[string]string{
	"":                  "No pattern filter; counts refer only to retained evidence.",
	"errors":            "At least one owned message has isError=true, stopReason=error, or a producer return has status=error.",
	"interruptions":     "At least one owned message has stopReason=aborted or a producer return has status=stopped.",
	"repeated-tools":    "At least one exact case-sensitive tool name occurs in two or more owned tool-call blocks; count is occurrences beyond the first per name.",
	"repeated-commands": "At least one exact command argument string occurs in two or more owned tool-call blocks; count is occurrences beyond the first per string. No shell normalization.",
	"timing-gaps":       "Maximum gap between sorted known owned work timestamps is at least 300 seconds; no causal attribution.",
	"missing-evidence":  "Work records unavailable, any work timestamp unknown, or no authenticated launch retained.",
}

func newTranscriptReviewCmd() *cobra.Command {
	var queue, order, pattern, cursor string
	var limit int
	var asJSON, payload bool
	var last, page int
	var snapshot string
	var text bundleTextOptions
	c := &cobra.Command{Use: "review <session>", Short: "Review retained open-handoff, ambiguous-handoff, or archive evidence (not liveness)", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
		if err := text.validate(c); err != nil {
			return err
		}
		if text.Format == "text" {
			if !c.Flags().Changed("last") {
				return fmt.Errorf("review: text requires --last")
			}
			payload = true
		}
		if !asJSON && text.Format != "text" {
			return fmt.Errorf("review: --json is required")
		}
		if c.Flags().Changed("last") {
			p, err := buildReviewTails(args[0], queue, order, pattern, limit, cursor, last, payload, page, snapshot)
			if err != nil {
				return err
			}
			if text.Format == "text" {
				return renderBundleText(c.OutOrStdout(), p, text, func(n int) (ledgerPage, error) {
					return buildReviewTails(args[0], queue, order, pattern, limit, cursor, last, payload, n, p.Snapshot)
				})
			}
			return json.NewEncoder(c.OutOrStdout()).Encode(p)
		}
		for _, flag := range []string{"payload", "page", "snapshot"} {
			if c.Flags().Changed(flag) {
				return fmt.Errorf("review: --%s requires --last", flag)
			}
		}
		p, err := buildReviewPage(args[0], queue, order, pattern, limit, cursor)
		if err != nil {
			return err
		}
		return json.NewEncoder(c.OutOrStdout()).Encode(p)
	}}
	addBundleTextFlags(c, &text)
	c.Flags().IntVar(&last, "last", 0, "bundle the latest 1–200 events per selected room")
	c.Flags().BoolVar(&payload, "payload", false, "include native tail payloads (requires --last)")
	c.Flags().IntVar(&page, "page", 1, "bundle transport page (requires --last)")
	c.Flags().StringVar(&snapshot, "snapshot", "", "bundle snapshot for later transport pages (requires --last)")
	c.Flags().BoolVar(&asJSON, "json", false, "emit bounded deterministic review JSON")
	c.Flags().StringVar(&queue, "queue", "open-handoff", "open-handoff, ambiguous-handoff, or archive (all retained non-ROOT rooms)")
	c.Flags().StringVar(&order, "order", "observed-recent", "observed-recent or canonical")
	c.Flags().StringVar(&pattern, "pattern", "", "errors, interruptions, repeated-tools, repeated-commands, timing-gaps, or missing-evidence")
	c.Flags().IntVar(&limit, "limit", 20, "rows per page (1–200)")
	c.Flags().StringVar(&cursor, "cursor", "", "continue the exact frozen review selection")
	return c
}

func reviewEligible(r reviewRow, q string) bool {
	switch q {
	case "open-handoff":
		return r.AuthenticatedLaunches > 0 && r.Terminals == 0 && r.Returns == 0
	case "ambiguous-handoff":
		return r.Launches > 0 && r.Returns > 0 // No recorded attempt pairing exists in Pi handoff records.
	case "archive":
		return true
	}
	return false
}

func reviewPattern(r reviewRow, p string) bool {
	switch p {
	case "errors":
		return r.Errors > 0
	case "interruptions":
		return r.Interruptions > 0
	case "repeated-tools":
		return r.RepeatedTools > 0
	case "repeated-commands":
		return r.RepeatedCommands > 0
	case "timing-gaps":
		return r.MaxGapSeconds >= 300
	case "missing-evidence":
		return r.WorkStatus == "unavailable" || r.UnknownTimestamps > 0 || r.AuthenticatedLaunches == 0
	}
	return true
}

func reviewTimestamp(r rawRecord, m map[string]json.RawMessage) (time.Time, bool) {
	if t, e := time.Parse(time.RFC3339Nano, r.Timestamp); e == nil {
		return t, true
	}
	var s string
	if json.Unmarshal(m["timestamp"], &s) == nil {
		if t, e := time.Parse(time.RFC3339Nano, s); e == nil {
			return t, true
		}
	}
	var millis float64
	if len(m["timestamp"]) > 0 && string(m["timestamp"]) != "null" && json.Unmarshal(m["timestamp"], &millis) == nil && millis > 0 {
		return time.UnixMilli(int64(millis)).UTC(), true
	}
	return time.Time{}, false
}

func reviewLabel(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	rr := []rune(s)
	if len(rr) > 120 {
		return string(rr[:117]) + "..."
	}
	return s
}

func buildReviewPage(session, queue, order, pattern string, limit int, cursor string) (reviewPage, error) {
	c, e := reviewCapture(session, cursor)
	if e != nil {
		return reviewPage{}, e
	}
	return buildReviewPageCaptured(c, queue, order, pattern, limit, cursor)
}

func reviewCapture(session, cursor string) (*transcriptCapture, error) {
	if cursor == "" {
		return newTranscriptCapture(session)
	}
	b, e := base64.RawURLEncoding.DecodeString(cursor)
	var token reviewCursor
	if e != nil || json.Unmarshal(b, &token) != nil || token.Version != 1 {
		return nil, fmt.Errorf("review: invalid cursor")
	}
	c, e := loadTranscriptCapture(token.Capture)
	if e != nil {
		return nil, e
	}
	path, e := filepath.Abs(session)
	if e != nil {
		return nil, e
	}
	if path != c.Path && path != c.InputPath {
		return nil, fmt.Errorf("review: capture path mismatch")
	}
	return c, nil
}

func buildReviewPageCaptured(c *transcriptCapture, queue, order, pattern string, limit int, cursor string) (reviewPage, error) {
	var empty reviewPage
	if queue != "open-handoff" && queue != "ambiguous-handoff" && queue != "archive" {
		return empty, fmt.Errorf("review: invalid queue %q", queue)
	}
	if order != "observed-recent" && order != "canonical" {
		return empty, fmt.Errorf("review: invalid order %q", order)
	}
	algorithm, ok := reviewAlgorithms[pattern]
	if !ok {
		return empty, fmt.Errorf("review: invalid pattern %q", pattern)
	}
	if limit < 1 || limit > 200 {
		return empty, fmt.Errorf("review: limit must be between 1 and 200")
	}
	path := c.Path
	records, err := c.read(path)
	if err != nil {
		return empty, err
	}
	agents, err := buildPiTreeUsing(path, c.read, c.resolve)
	if err != nil {
		return empty, err
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i].ID < agents[j].ID })
	h := sha256.New()
	enc := json.NewEncoder(h)
	_ = enc.Encode(struct {
		Version                     int
		Path, Queue, Order, Pattern string
		Limit                       int
	}{1, path, queue, order, pattern, limit})
	_ = enc.Encode(c.ID)
	_ = enc.Encode(records)
	_ = enc.Encode(agents)
	byID := map[string]*reviewRow{}
	for _, a := range agents {
		if a.ID == "ROOT" {
			continue
		}
		byID[a.ID] = &reviewRow{ID: a.ID, Parentage: a.ParentageStatus, Label: "Untitled room · " + reviewShortID(a.ID), LabelProvenance: "untitled", LabelEventID: "unavailable", Pairing: "not_inferred", LastObservedKind: "unknown", LastObservedEventID: "unavailable", RecencyBasis: "work", Liveness: "not_inferred", WorkStatus: "unavailable"}
	}
	for _, handoff := range piHandoffs(records) {
		r := byID[handoff.Child]
		if r == nil {
			continue
		}
		if handoff.Kind == "launch" {
			r.Launches++
			if handoff.Match == "matched" {
				r.AuthenticatedLaunches++
			}
			if handoff.Description != "" {
				r.Label = reviewLabel(handoff.Description)
				r.LabelProvenance = "recorded"
				r.LabelEventID = ledgerID(path, handoff.Record.RecordOrdinal, r.ID, "handoff:launch")
				r.LaunchOccurrence = r.Launches
			}
		} else {
			r.Returns++
			r.Terminals++ // Every producer terminal record counts, including unfamiliar statuses.
			var d piCustomData
			_ = json.Unmarshal(handoff.Record.Data, &d)

			if d.Status == "error" {
				r.Errors++
			}
			if d.Status == "stopped" {
				r.Interruptions++
			}
		}
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	rows := []reviewRow{}
	unknown := 0
	for _, id := range ids {
		r := byID[id]
		recs, _ := c.ledger(id)
		_ = enc.Encode(recs)
		tools, commands := map[string]int{}, map[string]int{}
		stamps := []time.Time{}
		for _, lr := range recs {
			if lr.Lifecycle {
				continue
			}
			var m map[string]json.RawMessage
			if json.Unmarshal(lr.Record.Message, &m) != nil {
				continue
			}
			role := ledgerString(m, "role")
			if role == "" {
				role = lr.Record.Type
			}
			var blocks []map[string]json.RawMessage
			_ = json.Unmarshal(m["content"], &blocks)
			eventID := ledgerID(lr.Path, lr.Record.RecordOrdinal, id, "message")
			if role == "user" && r.LabelProvenance == "untitled" {
				var text string
				_ = json.Unmarshal(m["content"], &text)
				for _, b := range blocks {
					if ledgerString(b, "type") == "text" {
						text += " " + ledgerString(b, "text")
					}
				}
				if strings.TrimSpace(text) != "" {
					r.Label = reviewLabel(text)
					r.LabelProvenance = "opening"
					r.LabelEventID = eventID
				}
			}
			if role != "assistant" && role != "toolResult" && role != "tool" {
				continue
			}
			r.WorkStatus = "observed"
			kind := "assistant"
			if role != "assistant" {
				kind = "tool_result"
			}
			if string(m["isError"]) == "true" || ledgerString(m, "stopReason") == "error" {
				r.Errors++
			}
			if ledgerString(m, "stopReason") == "aborted" {
				r.Interruptions++
			}
			if role == "assistant" {
				for _, b := range blocks {
					typ := ledgerString(b, "type")
					if typ != "toolCall" && typ != "tool_use" {
						continue
					}
					kind = "tool_call"
					name := ledgerString(b, "name")
					if name != "" {
						tools[name]++
					}
					args := b["arguments"]
					if len(args) == 0 {
						args = b["input"]
					}
					var a map[string]json.RawMessage
					_ = json.Unmarshal(args, &a)
					if command := ledgerString(a, "command"); command != "" {
						commands[command]++
					}
				}
			}
			stamp, known := reviewTimestamp(lr.Record, m)
			if !known {
				r.UnknownTimestamps++
				continue
			}
			stamps = append(stamps, stamp)
			previous := time.Time{}
			if r.LastObservedAt != nil {
				previous, _ = time.Parse(time.RFC3339Nano, *r.LastObservedAt)
			}
			if r.LastObservedAt == nil || stamp.After(previous) {
				s := stamp.UTC().Format(time.RFC3339Nano)
				r.LastObservedAt = &s
				r.LastObservedKind = kind
				r.LastObservedEventID = eventID
			}
		}
		for _, n := range tools {
			if n > 1 {
				r.RepeatedTools += n - 1
			}
		}
		for _, n := range commands {
			if n > 1 {
				r.RepeatedCommands += n - 1
			}
		}
		sort.Slice(stamps, func(i, j int) bool { return stamps[i].Before(stamps[j]) })
		for i := 1; i < len(stamps); i++ {
			if d := stamps[i].Sub(stamps[i-1]).Seconds(); d > r.MaxGapSeconds {
				r.MaxGapSeconds = d
			}
		}
		if r.WorkStatus == "unavailable" || r.UnknownTimestamps > 0 || r.AuthenticatedLaunches == 0 {
			unknown++
		}
		if reviewEligible(*r, queue) && reviewPattern(*r, pattern) {
			rows = append(rows, *r)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if order == "observed-recent" {
			a, b := rows[i].LastObservedAt, rows[j].LastObservedAt
			if a == nil && b != nil {
				return false
			}
			if a != nil && b == nil {
				return true
			}
			if a != nil && b != nil && *a != *b {
				ta, _ := time.Parse(time.RFC3339Nano, *a)
				tb, _ := time.Parse(time.RFC3339Nano, *b)
				return ta.After(tb)
			}
		}
		return rows[i].ID < rows[j].ID
	})
	_ = enc.Encode(rows)
	snapshot := hex.EncodeToString(h.Sum(nil))
	offset := 0
	if cursor != "" {
		b, e := base64.RawURLEncoding.DecodeString(cursor)
		var c reviewCursor
		if e != nil || json.Unmarshal(b, &c) != nil || c.Version != 1 || c.Snapshot != snapshot || c.Offset < 1 || c.Offset >= len(rows) {
			return empty, fmt.Errorf("review: stale or mismatched cursor")
		}
		offset = c.Offset
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	p := reviewPage{Path: path, Snapshot: snapshot, CaptureID: c.ID, Queue: queue, Order: order, Pattern: pattern, Algorithm: algorithm, Population: len(ids), Eligible: len(rows), Offset: offset, Returned: end - offset, Omitted: len(rows) - end, Unknown: unknown, Rows: rows[offset:end]}
	if end < len(rows) {
		b, _ := json.Marshal(reviewCursor{1, snapshot, end, c.ID})
		p.NextCursor = base64.RawURLEncoding.EncodeToString(b)
	}
	return p, nil
}

func reviewShortID(id string) string {
	r := []rune(id)
	if len(r) > 12 {
		return string(r[:12])
	}
	return id
}
