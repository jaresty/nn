package cmd

import (
	"encoding/json"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

type childDetailResolution struct {
	DetailSource   string          `json:"detail_source,omitempty"`
	Custody        string          `json:"custody,omitempty"`
	JoinEvidence   map[string]bool `json:"join_evidence,omitempty"`
	CandidateCount int             `json:"candidate_count,omitempty"`
	ResolvedPath   string          `json:"resolved_path,omitempty"`
}

type piOwnedSessionCandidate struct {
	Path    string
	Records []rawRecord
}

func piLaunchForChild(recs []rawRecord, id string) (piHandoff, string, string, bool) {
	for _, h := range piHandoffs(recs) {
		if h.Kind != "launch" || h.Child != id || h.Match != "matched" || h.Invocation == nil {
			continue
		}
		var block map[string]json.RawMessage
		if json.Unmarshal(h.Invocation.Raw, &block) != nil {
			continue
		}
		args := block["arguments"]
		if len(args) == 0 {
			args = block["input"]
		}
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(args, &fields)
		typ := ledgerString(fields, "subagent_type")
		if typ == "" {
			typ = ledgerString(fields, "subagentType")
		}
		return h, typ, h.Invocation.Record.Timestamp, typ != "" && h.Description != ""
	}
	return piHandoff{}, "", "", false
}

func piSessionName(recs []rawRecord) string {
	for _, r := range recs {
		if r.Type != "custom" || r.CustomType != "session_info" {
			continue
		}
		var fields map[string]json.RawMessage
		if json.Unmarshal(r.Data, &fields) == nil {
			return ledgerString(fields, "name")
		}
	}
	return ""
}

func piOpeningAssignment(recs []rawRecord) string {
	for _, r := range recs {
		if r.Type != "message" {
			continue
		}
		var msg map[string]json.RawMessage
		if json.Unmarshal(r.Message, &msg) == nil && ledgerString(msg, "role") == "user" {
			return strings.TrimSpace(ledgerText(msg["content"]))
		}
	}
	return ""
}

func canonicalPath(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return ""
	}
	return filepath.Clean(resolved)
}

func piOwnedSessionCandidates(parent string, recs []rawRecord, id string) ([]piOwnedSessionCandidate, map[string]bool) {
	handoff, agentType, launched, ok := piLaunchForChild(recs, id)
	evidence := map[string]bool{"parent_session": false, "agent_type": false, "agent_id_prefix": false, "assignment": false, "launch_time": false}
	if !ok {
		return nil, evidence
	}
	parent = canonicalPath(parent)
	launchTime, launchErr := time.Parse(time.RFC3339Nano, launched)
	var candidates []piOwnedSessionCandidate
	_ = filepath.WalkDir(filepath.Dir(parent), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".jsonl" || canonicalPath(path) == parent {
			return nil
		}
		header, headerOK := readPiSessionHeader(path)
		if !headerOK || header.ParentSession == "" {
			return nil
		}
		recorded := header.ParentSession
		if !filepath.IsAbs(recorded) {
			recorded = filepath.Join(filepath.Dir(path), recorded)
		}
		if canonicalPath(recorded) != parent {
			return nil
		}
		rows, readErr := readRecords(path)
		if readErr != nil {
			return nil
		}
		name := piSessionName(rows)
		prefix := agentType + "#"
		if !strings.HasPrefix(name, prefix) || !strings.HasPrefix(id, strings.TrimPrefix(name, prefix)) {
			return nil
		}
		if piOpeningAssignment(rows) != handoff.Description {
			return nil
		}
		if launchErr == nil {
			first := ""
			for _, row := range rows {
				if row.Timestamp != "" {
					first = row.Timestamp
					break
				}
			}
			candidateTime, candidateErr := time.Parse(time.RFC3339Nano, first)
			if candidateErr != nil || candidateTime.Before(launchTime) {
				return nil
			}
		} else {
			return nil
		}
		candidates = append(candidates, piOwnedSessionCandidate{Path: canonicalPath(path), Records: ownedPiRecords(rows, "ROOT", false)})
		return nil
	})
	if len(candidates) > 0 {
		for key := range evidence {
			evidence[key] = true
		}
	}
	return candidates, evidence
}

func transcriptChildDetailResolution(path, id, schema, detail string) childDetailResolution {
	if schema != schemaPi || id == "" || id == "ROOT" {
		return childDetailResolution{}
	}
	canonical := canonicalPath(path)
	recs, err := readRecords(canonical)
	if err != nil {
		return childDetailResolution{DetailSource: "unavailable"}
	}
	if len(ownedPiRecords(recs, id, false)) > 0 {
		return childDetailResolution{DetailSource: "inline_detail", Custody: "producer_owned", CandidateCount: 1, ResolvedPath: canonical}
	}
	for _, loc := range piBackgroundLocators(recs) {
		if loc.AgentID == id {
			if _, safe := readOwnedPiSidechain(loc.Path, id); safe != "" {
				return childDetailResolution{DetailSource: "producer_locator", Custody: "producer_owned", CandidateCount: 1, ResolvedPath: safe}
			}
		}
	}
	candidates, evidence := piOwnedSessionCandidates(canonical, recs, id)
	if len(candidates) == 1 && detail == "available" {
		return childDetailResolution{DetailSource: "owned_session_fallback", Custody: "qualified_unique_join", JoinEvidence: evidence, CandidateCount: 1, ResolvedPath: candidates[0].Path}
	}
	return childDetailResolution{DetailSource: "unavailable", JoinEvidence: evidence, CandidateCount: len(candidates)}
}
