package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode"
)

type transcriptSearchContext struct {
	SourcePath string                         `json:"source_path"`
	AgentID    string                         `json:"agent_id"`
	Snapshot   string                         `json:"window_snapshot"`
	Query      *ledgerQueryReceipt            `json:"query"`
	Events     []transcriptSearchContextEvent `json:"events"`
}

type transcriptSearchContextEvent struct {
	EventID             string `json:"event_id"`
	Ordinal             int    `json:"ordinal"`
	Kind                string `json:"kind"`
	Role                string `json:"role,omitempty"`
	Match               bool   `json:"match"`
	WindowStart         bool   `json:"window_start"`
	Text                string `json:"text"`
	TruncatedCharacters int    `json:"truncated_characters"`
}

func addTranscriptSearchContext(result *transcriptSearchResult, options ledgerWindowOptions) error {
	type group struct {
		path, agent string
		matches     []int
	}
	groups := []group{}
	positions := map[string]int{}
	for i, m := range result.Matches {
		if runes := []rune(m.Excerpt); len(runes) > 1000 {
			result.Matches[i].Excerpt = string(runes[:1000])
			result.Matches[i].ExcerptTruncatedCharacters = len(runes) - 1000
		}
		key := m.SourcePath + "\x00" + m.AgentID
		j, ok := positions[key]
		if !ok {
			j = len(groups)
			positions[key] = j
			groups = append(groups, group{path: m.SourcePath, agent: m.AgentID})
		}
		groups[j].matches = append(groups[j].matches, i)
	}
	total := 0
	for _, g := range groups {
		records, schema, detail, err := ledgerRecords(g.path, g.agent)
		if err != nil {
			return fmt.Errorf("search context %s %s: %w", g.path, g.agent, err)
		}
		canonical, err := filepath.EvalSymlinks(g.path)
		if err != nil {
			return err
		}
		canonical, err = filepath.Abs(canonical)
		if err != nil {
			return err
		}
		o := options
		o.MaxMatches = 200
		o.AnchorIDs = nil
		for _, i := range g.matches {
			m := &result.Matches[i]
			found := false
			for _, r := range records {
				if r.Lifecycle || r.Record.RecordOrdinal != m.recordOrdinal {
					continue
				}
				source, e := filepath.EvalSymlinks(r.Path)
				if e != nil {
					return e
				}
				source, e = filepath.Abs(source)
				if e != nil {
					return e
				}
				if source != canonical {
					continue
				}
				nativeID := r.Record.ID
				if nativeID == "" {
					nativeID = fmt.Sprintf("record:%d", r.Record.RecordOrdinal)
				}
				if nativeID != m.EventID || r.Record.Timestamp != m.Timestamp || sha256.Sum256(r.Record.Message) != m.messageDigest {
					return fmt.Errorf("search: source changed before context acquisition")
				}
				m.ContextEventID = ledgerID(r.Path, r.Record.RecordOrdinal, g.agent, "message")
				o.AnchorIDs = append(o.AnchorIDs, m.ContextEventID)
				found = true
				break
			}
			if !found {
				return fmt.Errorf("search: attributable context anchor unavailable for %s %s", m.SourcePath, m.EventID)
			}
		}
		fields, _ := ledgerSelect("identity,message,tools,lifecycle")
		events, err := projectLedger(records, g.agent, fields, true)
		if err != nil {
			return err
		}
		p, err := buildWindowLedgerPage(g.path, g.agent, schema, detail, fields, true, events, 1, "", "", true, ledgerQuery{}, o)
		if err != nil {
			return err
		}
		total += len(p.Events)
		if total > 2000 {
			return fmt.Errorf("search: context exceeds 2000 events; reduce --limit or context")
		}
		context := transcriptSearchContext{SourcePath: g.path, AgentID: g.agent, Snapshot: p.Snapshot, Query: p.Query, Events: []transcriptSearchContextEvent{}}
		for _, raw := range p.Events {
			var event ledgerEvent
			if err = json.Unmarshal(raw, &event); err != nil {
				return err
			}
			id, _ := event["event_id"].(string)
			kind, _ := event["kind"].(string)
			ordinal, _ := event["ordinal"].(float64)
			match, _ := event["window_match"].(bool)
			start, _ := event["window_start"].(bool)
			role := ""
			if msg, ok := event["message"].(map[string]any); ok {
				role, _ = msg["role"].(string)
			}
			text := ledgerSearchText(event)
			text = strings.Join(strings.Fields(strings.Map(func(r rune) rune {
				if unicode.IsControl(r) {
					return ' '
				}
				return r
			}, text)), " ")
			runes := []rune(text)
			clipped := 0
			if len(runes) > 1000 {
				clipped = len(runes) - 1000
				text = string(runes[:1000])
			}
			context.Events = append(context.Events, transcriptSearchContextEvent{EventID: id, Ordinal: int(ordinal), Kind: kind, Role: role, Match: match, WindowStart: start, Text: text, TruncatedCharacters: clipped})
		}
		result.Context = append(result.Context, context)
	}
	return nil
}

func renderTranscriptSearchContext(w io.Writer, r transcriptSearchResult, asJSON bool) error {
	var out bytes.Buffer
	if asJSON {
		enc := json.NewEncoder(&out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(r); err != nil {
			return err
		}
		if out.Len() > 1048576 {
			return fmt.Errorf("search: context JSON exceeds 1 MiB; reduce --limit or context")
		}
	} else {
		fmt.Fprintf(&out, "matches: %d · truncated matches: %t · skipped files: %d\n", r.Returned, r.Truncated, r.SkippedFiles)
		for _, c := range r.Context {
			fmt.Fprintf(&out, "\n%s · %s · window snapshot: %s\n", c.SourcePath, c.AgentID, c.Snapshot)
			fmt.Fprintf(&out, "selected events: %d · omitted events: %d\n", len(c.Events), c.Query.Total-len(c.Events))
			previous := 0
			for _, e := range c.Events {
				if previous > 0 && e.Ordinal > previous+1 {
					fmt.Fprintf(&out, "-- %d events omitted --\n", e.Ordinal-previous-1)
				}
				previous = e.Ordinal
				marker := "CONTEXT"
				if e.Match {
					marker = "MATCH"
				}
				label := e.Kind
				if e.Role != "" {
					label = e.Role
				}
				fmt.Fprintf(&out, "%s %s [%s]: %s", marker, label, e.EventID, e.Text)
				if e.TruncatedCharacters > 0 {
					fmt.Fprintf(&out, " [truncated %d chars]", e.TruncatedCharacters)
				}
				fmt.Fprintln(&out)
			}
		}
		if out.Len() > 200000 {
			return fmt.Errorf("search: context text exceeds 200000 bytes; reduce --limit or context")
		}
	}
	_, err := w.Write(out.Bytes())
	return err
}
