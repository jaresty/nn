package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// ownedPiRecords is shared by human-readable show and the structured ledger.
// A resolved sidechain requires explicit ownership even when requesting ROOT.
func ownedPiRecords(recs []rawRecord, id string, sidechain bool) []rawRecord {
	var out []rawRecord
	for _, r := range recs {
		if !isPiEventRecord(r) {
			continue
		}
		owner := r.AgentID
		if !sidechain && owner == "" {
			owner = "ROOT"
		}
		if owner == id && id != "" {
			out = append(out, r)
		}
	}
	return out
}

func readOwnedPiSidechain(path, id string) ([]rawRecord, string) {
	safe := validatePiSidechainPath(path, id)
	if safe == "" {
		return nil, ""
	}
	recs, err := readRecords(safe)
	if err != nil {
		return nil, ""
	}
	return ownedPiRecords(recs, id, true), safe
}

type ledgerRecord struct {
	Record    rawRecord
	Path      string
	Lifecycle bool
}

func ledgerRecords(session, id string) ([]ledgerRecord, string, string, error) {
	schema := classifyTranscript(session)
	path, err := filepath.Abs(session)
	if err != nil {
		return nil, "", "", err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return nil, "", "", err
	}
	recs, err := readRecords(path)
	if err != nil {
		return nil, "", "", err
	}
	var selected []rawRecord
	selectedPath := path
	var result []ledgerRecord
	switch schema {
	case schemaPi:
		selected = ownedPiRecords(recs, id, false)
		if len(selected) == 0 {
			for _, loc := range piBackgroundLocators(recs) {
				if loc.AgentID == id {
					selected, selectedPath = readOwnedPiSidechain(loc.Path, id)
					break
				}
			}
		}
		for _, r := range recs {
			if r.Type != "custom" || r.CustomType != "subagents:record" {
				continue
			}
			var d piCustomData
			if json.Unmarshal(r.Data, &d) == nil && d.ID == id {
				result = append(result, ledgerRecord{r, path, true})
			}
		}
	case schemaSDKCLI:
		if id != "ROOT" {
			if id == "" || strings.ContainsAny(id, "/\\") || id == "." || id == ".." {
				return nil, "", "", fmt.Errorf("events: invalid SDK agent id")
			}
			dir := strings.TrimSuffix(path, ".jsonl") + "/subagents"
			dir, err = filepath.EvalSymlinks(dir)
			if err != nil {
				return nil, "", "", err
			}
			child, e := filepath.EvalSymlinks(filepath.Join(dir, "agent-"+id+".jsonl"))
			if e != nil {
				return []ledgerRecord{}, schema, "unavailable", nil
			}
			if child != filepath.Join(dir, "agent-"+id+".jsonl") {
				return nil, "", "", fmt.Errorf("events: SDK child escapes subagents directory")
			}
			selectedPath = child
			recs, err = readRecords(child)
			if err != nil {
				return nil, "", "", err
			}
		}
		for _, r := range recs {
			if isPiEventRecord(r) && (r.AgentID == "" || r.AgentID == id) {
				selected = append(selected, r)
			}
		}
	case schemaClaudeCode:
		if id == "ROOT" {
			selected = ownedPiRecords(recs, id, false)
		}
	default:
		return nil, "", "", fmt.Errorf("events: unknown transcript schema")
	}
	for _, r := range selected {
		result = append(result, ledgerRecord{r, selectedPath, false})
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Path != result[j].Path {
			return result[i].Path < result[j].Path
		}
		return result[i].Record.RecordOrdinal < result[j].Record.RecordOrdinal
	})
	status := "unavailable"
	for _, r := range selected {
		var msg map[string]json.RawMessage
		if json.Unmarshal(r.Message, &msg) == nil && msg != nil {
			status = "available"
			break
		}
	}
	return result, schema, status, nil
}
