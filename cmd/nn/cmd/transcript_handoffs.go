package cmd

import "encoding/json"

type handoffReceipt struct {
	At          string `json:"at"`
	Status      string `json:"status"`
	Launches    int    `json:"launches"`
	Returns     int    `json:"returns"`
	Occurrences int    `json:"occurrences"`
	Pairing     string `json:"pairing"`
}

type piInvocation struct {
	Record      rawRecord
	Slot        int
	Raw         json.RawMessage
	Description string
}

type piHandoff struct {
	Record      rawRecord
	Child       string
	Owner       string
	Kind        string
	Description string
	CallID      string
	Match       string
	Invocation  *piInvocation
}

func handoffOwner(r rawRecord) string {
	if r.AgentID != "" {
		return r.AgentID
	}
	return "ROOT"
}

// Join only exact, unique Agent call IDs in the same recorded owner scope.
// ParentId is event sequencing, not a substitute for call identity.
func piHandoffs(recs []rawRecord) []piHandoff {
	calls := map[[2]string][]piInvocation{}
	acks := map[[2]string]int{}
	out := []piHandoff{}
	for _, r := range recs {
		owner := handoffOwner(r)
		if r.Type == "custom" && r.CustomType == "subagents:record" {
			var d piCustomData
			if json.Unmarshal(r.Data, &d) == nil && d.ID != "" {
				out = append(out, piHandoff{Record: r, Child: d.ID, Owner: owner, Kind: "return"})
			}
			continue
		}
		if !isPiEventRecord(r) {
			continue
		}
		var m map[string]json.RawMessage
		if json.Unmarshal(r.Message, &m) != nil {
			continue
		}
		role := ledgerString(m, "role")
		if role == "" {
			role = r.Type
		}
		if role == "assistant" {
			var blocks []json.RawMessage
			_ = json.Unmarshal(m["content"], &blocks)
			for i, raw := range blocks {
				var b map[string]json.RawMessage
				_ = json.Unmarshal(raw, &b)
				kind := ledgerString(b, "type")
				if (kind != "toolCall" && kind != "tool_use") || ledgerString(b, "name") != "Agent" {
					continue
				}
				id := ledgerString(b, "id")
				if id == "" {
					continue
				}
				args := b["arguments"]
				if len(args) == 0 {
					args = b["input"]
				}
				var a map[string]json.RawMessage
				_ = json.Unmarshal(args, &a)
				key := [2]string{owner, id}
				calls[key] = append(calls[key], piInvocation{r, i, raw, ledgerString(a, "description")})
			}
		}
		if role != "toolResult" || ledgerString(m, "toolName") != "Agent" {
			continue
		}
		var details map[string]json.RawMessage
		_ = json.Unmarshal(m["details"], &details)
		child := ledgerString(details, "agentId")
		if ledgerString(details, "status") != "background" || child == "" {
			continue
		}
		callID := ledgerString(m, "toolCallId")
		key := [2]string{owner, callID}
		if callID != "" {
			acks[key]++
		}
		out = append(out, piHandoff{Record: r, Child: child, Owner: owner, Kind: "launch", Description: ledgerString(details, "description"), CallID: callID, Match: "unavailable"})
	}
	for i := range out {
		h := &out[i]
		if h.Kind != "launch" || h.CallID == "" {
			continue
		}
		key := [2]string{h.Owner, h.CallID}
		matches := calls[key]
		switch {
		case len(matches) > 1 || acks[key] > 1:
			h.Match = "ambiguous"
		case len(matches) == 1:
			h.Match = "matched"
			inv := matches[0]
			h.Invocation = &inv
			if h.Description == "" {
				h.Description = inv.Description
			}
		default:
			h.Match = "missing"
		}
	}
	return out
}
