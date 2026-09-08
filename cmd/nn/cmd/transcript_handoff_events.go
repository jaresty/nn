package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

func buildHandoffPage(session, id, at string, selection []string, payload bool, page int, snapshot, eventFilter string, all bool) (ledgerPage, error) {
	if at != "launch" && at != "return" {
		return ledgerPage{}, fmt.Errorf("events: --at must be launch or return")
	}
	path, err := filepath.Abs(session)
	if err != nil {
		return ledgerPage{}, err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return ledgerPage{}, err
	}
	schema := classifyTranscript(path)
	receipt := &handoffReceipt{At: at, Status: "unavailable", Pairing: "not_inferred: occurrence numbers are independent per kind, not attempt pairs"}
	events := []ledgerEvent{}
	if schema != schemaPi {
		receipt.Status = "unsupported_schema"
		return buildLedgerPageWithHandoff(session, id, schema, "unavailable", selection, payload, events, page, snapshot, eventFilter, all, receipt)
	}
	recs, err := readRecords(path)
	if err != nil {
		return ledgerPage{}, err
	}
	known := id == "ROOT"
	for _, r := range recs {
		if r.AgentID == id {
			known = true
		}
	}
	handoffs := piHandoffs(recs)
	selected := []piHandoff{}
	for _, h := range handoffs {
		if h.Child == id {
			known = true
			selected = append(selected, h)
			if h.Kind == "launch" {
				receipt.Launches++
			} else {
				receipt.Returns++
			}
		}
	}
	if !known {
		return ledgerPage{}, fmt.Errorf("events: unknown agent %q in parent handoff evidence", id)
	}
	if receipt.Launches+receipt.Returns > 0 {
		receipt.Status = "not_observed"
	}
	keepLifecycle := false
	for _, s := range selection {
		if s == "lifecycle" {
			keepLifecycle = true
		}
	}
	occurrences := map[string]int{}
	for ordinal, h := range selected {
		occurrences[h.Kind]++
		if h.Kind != at {
			continue
		}
		receipt.Status = "observed"
		receipt.Occurrences++
		r := h.Record
		var stamp any
		origin := "unavailable"
		if r.Timestamp != "" {
			stamp = r.Timestamp
			origin = "record"
		} else if h.Kind == "launch" {
			var msg map[string]json.RawMessage
			_ = json.Unmarshal(r.Message, &msg)
			if raw := msg["timestamp"]; len(raw) > 0 && string(raw) != "null" {
				stamp = raw
				origin = "message"
			}
		}
		slot := "handoff:launch"
		if h.Kind == "return" {
			slot = "lifecycle"
		}
		nativeID := r.ID
		if nativeID == "" {
			nativeID = r.UUID
		}
		e := ledgerEvent{"event_id": ledgerID(path, r.RecordOrdinal, id, slot), "agent_id": id, "record_owner": h.Owner, "ordinal": ordinal + 1, "occurrence": occurrences[h.Kind], "kind": h.Kind, "timestamp": stamp, "timestamp_source": origin, "source": map[string]any{"path": path, "record_ordinal": r.RecordOrdinal, "record_id": nativeID}}
		life := map[string]any{"scope": "parent_handoff_record"}
		if h.Kind == "launch" {
			life["description"] = nil
			if h.Description != "" {
				life["description"] = h.Description
			}
			life["status"] = "background"
			life["call_id"] = h.CallID
			life["match_status"] = h.Match
			life["invocation"] = nil
			var invocation any
			if inv := h.Invocation; inv != nil {
				callRecordID := inv.Record.ID
				if callRecordID == "" {
					callRecordID = inv.Record.UUID
				}
				life["invocation"] = map[string]any{"event_id": ledgerID(path, inv.Record.RecordOrdinal, h.Owner, fmt.Sprintf("block:%d", inv.Slot)), "source": map[string]any{"path": path, "record_ordinal": inv.Record.RecordOrdinal, "record_id": callRecordID}, "tool": "Agent"}
				invocation = inv.Raw
			}
			if payload {
				e["payload"] = map[string]any{"acknowledgment": r.Message, "invocation": invocation}
			}
		} else {
			var d piCustomData
			_ = json.Unmarshal(r.Data, &d)
			life["status"] = d.Status
			life["started_at"] = nil
			life["completed_at"] = nil
			if d.StartedAt != 0 {
				life["started_at"] = d.StartedAt
			}
			if d.CompletedAt != 0 {
				life["completed_at"] = d.CompletedAt
			}
			if payload {
				e["payload"] = r.Data
			}
		}
		if keepLifecycle {
			e["lifecycle"] = life
		}
		events = append(events, e)
	}
	detail := "unavailable"
	if receipt.Status == "observed" {
		detail = "available"
	}
	return buildLedgerPageWithHandoff(session, id, schema, detail, selection, payload, events, page, snapshot, eventFilter, all, receipt)
}
