package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/spf13/cobra"
)

type bundleTextOptions struct {
	Format          string
	PerEvent, Total int
}

func addBundleTextFlags(c *cobra.Command, o *bundleTextOptions) {
	c.Flags().StringVar(&o.Format, "format", "json", "json or bounded readable text (automatically consumes transport pages)")
	c.Flags().IntVar(&o.PerEvent, "max-text-chars", 1000, "readable characters per evidence event, 1..10000 (text only)")
	c.Flags().IntVar(&o.Total, "max-output-chars", 24000, "total rendered characters, 2048..200000 (text only)")
}
func (o bundleTextOptions) validate(c *cobra.Command) error {
	if o.Format != "json" && o.Format != "text" {
		return fmt.Errorf("--format must be json or text")
	}
	if o.Format == "json" {
		for _, f := range []string{"max-text-chars", "max-output-chars"} {
			if c.Flags().Changed(f) {
				return fmt.Errorf("--%s requires --format text", f)
			}
		}
		return nil
	}
	if o.PerEvent < 1 || o.PerEvent > 10000 || o.Total < 2048 || o.Total > 200000 {
		return fmt.Errorf("text limits require max-text-chars 1..10000 and max-output-chars 2048..200000")
	}
	for _, f := range []string{"json", "payload", "page"} {
		if c.Flags().Changed(f) {
			return fmt.Errorf("text cannot combine with --%s", f)
		}
	}
	return nil
}

// Consume complete retained transport, including metadata fragments. Do not
// publish partial text if a later page fails integrity or segment validation.
func renderBundleText(w io.Writer, first ledgerPage, o bundleTextOptions, load func(int) (ledgerPage, error)) error {
	var output strings.Builder
	used, displayed, omitted, truncated := 0, 0, 0, 0
	emit := func(line string) bool {
		n := utf8.RuneCountInString(line)
		if used+n > o.Total-512 {
			return false
		}
		output.WriteString(line)
		used += n
		return true
	}
	emit(fmt.Sprintf("snapshot: %s\nReadable retained evidence (lossy; not an LLM summary).\n", first.Snapshot))
	render := func(raw json.RawMessage) error {
		var e map[string]json.RawMessage
		if err := json.Unmarshal(raw, &e); err != nil {
			return err
		}
		kind := ledgerString(e, "kind")
		val := func(key string) string {
			if b := e[key]; len(b) > 0 {
				return string(b)
			}
			return "unknown"
		}
		switch kind {
		case "review_receipt":
			emit(fmt.Sprintf("capture: %s · sequential source prefixes\nrooms retrieved: %s · eligible: %s · omitted rooms: %s · unavailable rooms: %s\nevents selected: %s · earlier events omitted: %s\n", ledgerString(e, "capture_id"), val("retrieved_rooms"), val("eligible"), val("omitted_rooms"), val("unavailable_rooms"), val("selected_events"), val("omitted_earlier_events")))
			if string(e["include_assignment"]) == "true" {
				emit("assignment records: " + val("selected_assignments") + " (independent launches; not additional recent events)\n")
			}
			if cursor := ledgerString(e, "next_room_cursor"); cursor != "" {
				emit("next room cursor: " + cursor + "\n")
			}
			return nil
		case "context_receipt":
			var recent map[string]json.RawMessage
			_ = json.Unmarshal(e["recent"], &recent)
			emit(fmt.Sprintf("capture: %s · sequential source prefixes\nroom: %s · steering: %s · governing attempt: %s\nrecent events selected: %s / %s retained\n", ledgerString(e, "capture_id"), cleanBundleText(ledgerString(e, "agent_id")), ledgerString(e, "steering_status"), ledgerString(e, "governing_attempt"), recent["selected_events"], recent["total_events"]))
			return nil
		case "review_room":
			var row map[string]json.RawMessage
			_ = json.Unmarshal(e["room"], &row)
			emit(fmt.Sprintf("\nROOM %s [%s] · detail: %s\n", cleanBundleText(ledgerString(row, "label")), cleanBundleText(ledgerString(e, "agent_id")), ledgerString(e, "detail_status")))
			if _, ok := e["assignment_events"]; ok {
				emit(fmt.Sprintf("assignment records: %s · steering: %s · governing attempt: %s\n", val("assignment_events"), ledgerString(e, "steering_status"), ledgerString(e, "governing_attempt")))
			}
			return nil
		}
		label, text := bundleEventText(e)
		text = cleanBundleText(text)
		rr := []rune(text)
		wasTruncated := len(rr) > o.PerEvent
		if wasTruncated {
			text = string(rr[:o.PerEvent]) + fmt.Sprintf(" [truncated %d chars]", len(rr)-o.PerEvent)
		}
		if text == "" {
			text = "[no readable text]"
		}
		line := fmt.Sprintf("%s [%s] room=%s: %s\n", cleanBundleText(label), ledgerString(e, "event_id"), cleanBundleText(ledgerString(e, "agent_id")), text)
		if emit(line) {
			displayed++
			if wasTruncated {
				truncated++
			}
		} else {
			omitted++
		}
		return nil
	}
	var fragment strings.Builder
	pendingID := ""
	nextSegment, totalSegments := 0, 0
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
			return fmt.Errorf("bundle transport identity mismatch")
		}
		for _, raw := range p.Events {
			var segment struct {
				EventID           string `json:"event_id"`
				Segment, Segments int
				Text              string
			}
			if err = json.Unmarshal(raw, &segment); err != nil {
				return err
			}
			if segment.Segment == 0 {
				if pendingID != "" {
					return fmt.Errorf("incomplete bundle fragment")
				}
				if err = render(raw); err != nil {
					return err
				}
				continue
			}
			if pendingID == "" {
				pendingID = segment.EventID
				nextSegment = 1
				totalSegments = segment.Segments
			}
			if segment.EventID != pendingID || segment.Segment != nextSegment || segment.Segments != totalSegments || totalSegments < 1 {
				return fmt.Errorf("invalid bundle segment order")
			}
			fragment.WriteString(segment.Text)
			nextSegment++
			if segment.Segment == totalSegments {
				if err = render(json.RawMessage(fragment.String())); err != nil {
					return err
				}
				fragment.Reset()
				pendingID = ""
			}
		}
	}
	if pendingID != "" {
		return fmt.Errorf("incomplete bundle transport")
	}
	fmt.Fprintf(&output, "\nTransport: %d/%d pages complete. Evidence events displayed: %d; omitted from display: %d; text-truncated: %d.\nMessage/tool records may represent the same operation. Use JSON for exact evidence.\n", first.Pages, first.Pages, displayed, omitted, truncated)
	if utf8.RuneCountInString(output.String()) > o.Total {
		return fmt.Errorf("text output exceeded budget")
	}
	_, err := io.WriteString(w, output.String())
	return err
}

func cleanBundleText(s string) string {
	return strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ' '
		}
		return r
	}, s)), " ")
}

func bundleEventText(e map[string]json.RawMessage) (string, string) {
	kind := ledgerString(e, "kind")
	var payload map[string]json.RawMessage
	_ = json.Unmarshal(e["payload"], &payload)
	switch kind {
	case "launch":
		var invocation map[string]json.RawMessage
		_ = json.Unmarshal(payload["invocation"], &invocation)
		args := invocation["arguments"]
		if len(args) == 0 {
			args = invocation["input"]
		}
		var a map[string]json.RawMessage
		_ = json.Unmarshal(args, &a)
		var life map[string]json.RawMessage
		_ = json.Unmarshal(e["lifecycle"], &life)
		prompt := ledgerString(a, "prompt")
		if prompt == "" {
			prompt = "[assignment unavailable]"
		}
		return "LAUNCH", fmt.Sprintf("occurrence %s · join: %s · %s · %s", e["occurrence"], ledgerString(life, "match_status"), ledgerString(life, "description"), prompt)
	case "tool_call":
		args := payload["arguments"]
		if len(args) == 0 {
			args = payload["input"]
		}
		var arguments map[string]json.RawMessage
		_ = json.Unmarshal(args, &arguments)
		if command := ledgerString(arguments, "command"); command != "" {
			return "CALL", ledgerString(payload, "name") + " " + command
		}
		return "CALL", ledgerString(payload, "name") + " " + string(args)
	case "tool_result":
		text := ledgerReadableContent(payload["content"])
		if string(payload["isError"]) == "true" || string(payload["is_error"]) == "true" {
			text = "[tool error] " + text
		}
		return "RESULT", text
	case "lifecycle":
		return "LIFECYCLE", ledgerString(payload, "status")
	default:
		role := ledgerString(payload, "role")
		var message map[string]json.RawMessage
		_ = json.Unmarshal(e["message"], &message)
		if role == "" {
			role = ledgerString(message, "role")
		}
		if role == "" {
			role = kind
		}
		text := ledgerReadableContent(payload["content"])
		stop, failure := ledgerString(message, "stop_reason"), ledgerString(message, "error_message")
		if stop == "error" || stop == "aborted" || failure != "" {
			text = fmt.Sprintf("[failure %s: %s] %s", stop, failure, text)
		}
		return strings.ToUpper(role), text
	}
}
