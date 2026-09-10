package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// renderLedgerText is a deterministic, explicitly lossy display, not a summary.
func renderLedgerText(w io.Writer, p ledgerPage, maxChars int) error {
	omitted := 0
	if p.Query != nil {
		omitted = p.Query.Total - len(p.Events)
	}
	if _, e := fmt.Fprintf(w, "snapshot: %s\nreturned: %d · omitted: %d · detail: %s\n", p.Snapshot, len(p.Events), omitted, p.DetailStatus); e != nil {
		return e
	}
	windowed := p.Query != nil && p.Query.Window != nil
	if windowed {
		r := p.Query.Window
		if _, err := fmt.Fprintf(w, "matches: %d of %d · context: %d · omitted matches: %d · windows: %d\n", r.SelectedMatches, r.Matches, r.ContextEvents, r.OmittedMatches, r.Windows); err != nil {
			return err
		}
	}
	previous := 0
	for _, raw := range p.Events {
		var event map[string]json.RawMessage
		if e := json.Unmarshal(raw, &event); e != nil {
			return e
		}
		if windowed {
			var ordinal int
			_ = json.Unmarshal(event["ordinal"], &ordinal)
			if previous > 0 && ordinal > previous+1 {
				if _, err := fmt.Fprintf(w, "-- %d events omitted --\n", ordinal-previous-1); err != nil {
					return err
				}
			}
			previous = ordinal
		}
		var payload map[string]json.RawMessage
		_ = json.Unmarshal(event["payload"], &payload)
		kind := ledgerString(event, "kind")
		label := strings.ToUpper(kind)
		text := ""
		switch kind {
		case "message":
			label = strings.ToUpper(ledgerString(payload, "role"))
			if label == "" {
				label = "MESSAGE"
			}
			text = ledgerReadableContent(payload["content"])
		case "tool_result":
			label = "RESULT"
			text = ledgerReadableContent(payload["content"])
		case "tool_call":
			label = "CALL"
			text = ledgerString(payload, "name")
			args := payload["arguments"]
			if len(args) == 0 {
				args = payload["input"]
			}
			var fields map[string]json.RawMessage
			_ = json.Unmarshal(args, &fields)
			value := ledgerString(fields, "command")
			if value == "" {
				value = ledgerString(fields, "path")
			}
			if value == "" && len(args) > 0 {
				value = string(args)
			}
			text += " " + value
		case "lifecycle":
			label = "LIFECYCLE"
			text = ledgerString(payload, "status")
		}
		// Render control characters as whitespace so retained text cannot emit terminal escapes.
		text = strings.Map(func(r rune) rune {
			if unicode.IsControl(r) {
				return ' '
			}
			return r
		}, text)
		text = strings.Join(strings.Fields(text), " ")
		runes := []rune(text)
		if len(runes) > maxChars {
			text = string(runes[:maxChars]) + fmt.Sprintf(" [truncated %d chars]", len(runes)-maxChars)
		}
		if text == "" {
			text = "[no readable text]"
		}
		if windowed {
			var match bool
			_ = json.Unmarshal(event["window_match"], &match)
			if match {
				label = "MATCH " + label
			} else {
				label = "CONTEXT " + label
			}
		}
		if _, e := fmt.Fprintf(w, "%s [%s]: %s\n", label, ledgerString(event, "event_id"), text); e != nil {
			return e
		}
	}
	return nil
}

func ledgerReadableContent(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	var blocks []map[string]json.RawMessage
	_ = json.Unmarshal(raw, &blocks)
	parts := []string{}
	for _, b := range blocks {
		kind := ledgerString(b, "type")
		if kind == "text" || kind == "output_text" {
			parts = append(parts, ledgerString(b, "text"))
		}
	}
	return strings.Join(parts, " ")
}
