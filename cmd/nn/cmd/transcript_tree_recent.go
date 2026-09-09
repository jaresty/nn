package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"time"
)

type treeChildRecency struct {
	LastObservedAt *string `json:"last_observed_at"`
	EventID        string  `json:"event_id"`
	Basis          string  `json:"basis"`
}

type retainedHallway struct {
	Request string
	Page    treeChildPage
}

// Retain the full ordered projection, not a live-source cursor. This shares the
// private, expiring, atomic capture store but never reopens raw capture on replay.
func buildOrderedTreeChildPage(session, parent, order, selected string, limit int, cursor string, strict bool) (treeChildPage, error) {
	empty := treeChildPage{}
	path, e := filepath.Abs(session)
	if e != nil {
		return empty, e
	}
	requestBytes, _ := json.Marshal([]any{path, parent, order, selected, limit, strict})
	request := string(requestBytes)
	var retained retainedHallway
	snapshot := ""
	start := 0
	if cursor != "" {
		b, e := base64.RawURLEncoding.DecodeString(cursor)
		var c treeChildCursor
		if e != nil || json.Unmarshal(b, &c) != nil || c.Version != 2 || c.Parent != parent || !validCaptureID(c.Snapshot) {
			return empty, fmt.Errorf("tree: stale or mismatched cursor")
		}
		b, e = readCaptureFile(c.Snapshot + ".hallway")
		if e != nil {
			return empty, fmt.Errorf("tree: retained hallway unavailable: %w", e)
		}
		if captureHash(b) != c.Snapshot || json.Unmarshal(b, &retained) != nil || retained.Request != request {
			return empty, fmt.Errorf("tree: stale or mismatched cursor or corrupt hallway")
		}
		if c.After < 0 || c.After >= len(retained.Page.Children) {
			return empty, fmt.Errorf("tree: invalid cursor position")
		}
		snapshot = c.Snapshot
		start = c.After + 1
	} else {
		c, e := newTranscriptCapture(session)
		if e != nil {
			return empty, e
		}
		agents, e := buildPiTreeUsing(c.Path, c.read, c.resolve)
		if e != nil {
			return empty, e
		}
		if strict {
			if e = validateTree(agents); e != nil {
				return empty, e
			}
		} else {
			repairTree(&agents)
			rollupSubtreeCost(agents)
		}
		p, e := buildTreeChildPage(agents, parent, 0, "")
		if e != nil {
			return empty, e
		}
		found := selected == ""
		stamps := map[string]time.Time{}
		for i := range p.Children {
			r := &p.Children[i]
			if r.ID == selected {
				found = true
			}
			r.Recency = &treeChildRecency{Basis: "work", EventID: "unavailable"}
			records, _ := c.ledger(r.ID)
			for _, lr := range records {
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
				if role != "assistant" && role != "toolResult" && role != "tool" {
					continue
				}
				stamp, known := reviewTimestamp(lr.Record, m)
				previous, seen := stamps[r.ID]
				if known && (!seen || stamp.After(previous)) {
					stamps[r.ID] = stamp
					s := stamp.UTC().Format(time.RFC3339Nano)
					r.Recency.LastObservedAt = &s
					r.Recency.EventID = ledgerID(lr.Path, lr.Record.RecordOrdinal, r.ID, "message")
				}
			}
		}
		if !found {
			return empty, fmt.Errorf("tree: selected room %q is not a direct child of %q", selected, parent)
		}
		sort.Slice(p.Children, func(i, j int) bool {
			a, b := p.Children[i], p.Children[j]
			if (a.ID == selected) != (b.ID == selected) {
				return a.ID == selected
			}
			if order == "observed-recent" {
				at, ak := stamps[a.ID]
				bt, bk := stamps[b.ID]
				if ak != bk {
					return ak
				}
				if ak && !at.Equal(bt) {
					return at.After(bt)
				}
			}
			return a.ID < b.ID
		})
		p.Order = order
		p.Selected = selected
		p.CaptureID = c.ID
		p.Snapshot = ""
		retained = retainedHallway{Request: request, Page: p}
		b, e := json.Marshal(retained)
		if e != nil {
			return empty, e
		}
		snapshot = captureHash(b)
		if e = writeCaptureFile(snapshot+".hallway", b); e != nil {
			return empty, e
		}
	}
	p := retained.Page
	end := len(p.Children)
	if limit > 0 && limit < end-start {
		end = start + limit
	}
	p.Children = p.Children[start:end]
	p.Snapshot = snapshot
	p.Returned = len(p.Children)
	p.Omitted = p.TotalChildren - end
	p.NextCursor = ""
	if end < p.TotalChildren {
		b, _ := json.Marshal(treeChildCursor{Version: 2, Snapshot: snapshot, Parent: parent, After: end - 1})
		p.NextCursor = base64.RawURLEncoding.EncodeToString(b)
	}
	return p, nil
}
