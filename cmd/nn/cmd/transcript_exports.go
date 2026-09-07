package cmd

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"
)

// Projection is deliberately downstream of complete tree validation and rollup.
func projectTranscriptTree(rows []agent, id, fields string) (any, error) {
	selected := rows
	if id != "" {
		selected = []agent{}
		for _, a := range rows {
			if a.ID == id {
				selected = append(selected, a)
			}
		}
		if len(selected) == 0 {
			return nil, fmt.Errorf("tree: unknown agent %q", id)
		}
	}
	if fields == "" {
		return selected, nil
	}
	allowed := map[string]bool{}
	typ := reflect.TypeOf(agent{})
	for i := 0; i < typ.NumField(); i++ {
		name := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			allowed[name] = true
		}
	}
	names := strings.Split(fields, ",")
	for i, name := range names {
		name = strings.TrimSpace(name)
		if !allowed[name] {
			return nil, fmt.Errorf("tree: unknown field %q", name)
		}
		names[i] = name
	}
	out := make([]map[string]json.RawMessage, 0, len(selected))
	for _, a := range selected {
		b, err := json.Marshal(a)
		if err != nil {
			return nil, err
		}
		var original map[string]json.RawMessage
		if err = json.Unmarshal(b, &original); err != nil {
			return nil, err
		}
		row := map[string]json.RawMessage{}
		for _, name := range names {
			value := original[name]
			if value == nil {
				value = json.RawMessage("null")
			}
			row[name] = value
		}
		out = append(out, row)
	}
	return out, nil
}

func completeTranscriptShow(session, id string, raw bool, text string) (any, error) {
	if !utf8.ValidString(text) {
		return nil, fmt.Errorf("transcript show: projected output is not valid UTF-8")
	}
	mode := "meaningful"
	if raw {
		mode = "raw"
	}
	snapshot, err := transcriptShowSnapshot(session, id, mode, text)
	if err != nil {
		return nil, err
	}
	return struct {
		All      bool   `json:"all"`
		Snapshot string `json:"snapshot"`
		Mode     string `json:"mode"`
		Text     string `json:"text"`
	}{true, snapshot, mode, text}, nil
}

func completeLedgerExport(page ledgerPage, events []json.RawMessage) ledgerPage {
	page.All = true
	page.Page = 1
	page.Pages = 1
	page.NextPage = 0
	page.Events = events
	if page.Events == nil {
		page.Events = []json.RawMessage{}
	}
	return page
}
