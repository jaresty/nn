package cmd

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
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

func piLaunchForChild(recs []rawRecord, id string) (piHandoff, string, string, string, bool) {
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
		assignment := ledgerString(fields, "prompt")
		return h, typ, assignment, h.Invocation.Record.Timestamp, typ != "" && assignment != ""
	}
	return piHandoff{}, "", "", "", false
}

func piSessionName(recs []rawRecord) string {
	for _, r := range recs {
		if r.Type == "session_info" && r.Name != "" {
			return r.Name
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
			return piDelegatedAssignment(strings.TrimSpace(ledgerText(msg["content"])))
		}
	}
	return ""
}

func piDelegatedAssignment(text string) string {
	const contextHeader = "# Parent Conversation Context\n"
	const taskBoundary = "\n---\n# Your Task (below)\n"
	if !strings.HasPrefix(text, contextHeader) {
		return text
	}
	boundary := strings.LastIndex(text, taskBoundary)
	if boundary < 0 {
		return text
	}
	return strings.TrimSpace(text[boundary+len(taskBoundary):])
}

type piOwnedSessionIdentity struct {
	Header         piSessionHeader
	Name           string
	Assignment     string
	FirstTimestamp string
}

func readPiOwnedSessionIdentity(path string) (piOwnedSessionIdentity, bool) {
	file, err := os.Open(path)
	if err != nil {
		return piOwnedSessionIdentity{}, false
	}
	defer file.Close()
	var identity piOwnedSessionIdentity
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		var record rawRecord
		if json.Unmarshal(line, &record) != nil {
			return piOwnedSessionIdentity{}, false
		}
		if identity.Header.ID == "" {
			if json.Unmarshal(line, &identity.Header) != nil || identity.Header.Type != "session" || identity.Header.ID == "" {
				return piOwnedSessionIdentity{}, false
			}
		}
		if identity.FirstTimestamp == "" && record.Timestamp != "" {
			identity.FirstTimestamp = record.Timestamp
		}
		if record.Type == "session_info" && record.Name != "" {
			identity.Name = record.Name
		}
		if record.Type == "message" {
			var msg map[string]json.RawMessage
			if json.Unmarshal(record.Message, &msg) == nil && ledgerString(msg, "role") == "user" {
				identity.Assignment = piDelegatedAssignment(strings.TrimSpace(ledgerText(msg["content"])))
				return identity, identity.Name != ""
			}
		}
	}
	return identity, false
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
	_, agentType, assignment, launched, ok := piLaunchForChild(recs, id)
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
		identity, identityOK := readPiOwnedSessionIdentity(path)
		if !identityOK || identity.Header.ParentSession == "" {
			return nil
		}
		recorded := identity.Header.ParentSession
		if !filepath.IsAbs(recorded) {
			recorded = filepath.Join(filepath.Dir(path), recorded)
		}
		if canonicalPath(recorded) != parent {
			return nil
		}
		prefix := agentType + "#"
		if !strings.HasPrefix(identity.Name, prefix) || !strings.HasPrefix(id, strings.TrimPrefix(identity.Name, prefix)) {
			return nil
		}
		if identity.Assignment != assignment {
			return nil
		}
		if launchErr == nil {
			candidateTime, candidateErr := time.Parse(time.RFC3339Nano, identity.FirstTimestamp)
			if candidateErr != nil || candidateTime.Before(launchTime) {
				return nil
			}
		} else {
			return nil
		}
		rows, readErr := readRecords(path)
		if readErr != nil {
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

func transcriptChildDetailResolution(path, id, schema, detail string, selectedPaths ...string) childDetailResolution {
	if schema != schemaPi || id == "" || id == "ROOT" {
		return childDetailResolution{}
	}
	canonical := canonicalPath(path)
	if detail == "available" && len(selectedPaths) > 0 {
		selected := canonicalPath(selectedPaths[0])
		switch {
		case selected == canonical:
			return childDetailResolution{DetailSource: "inline_detail", Custody: "producer_owned", CandidateCount: 1, ResolvedPath: canonical}
		case validatePiSidechainPath(selected, id) == selected:
			return childDetailResolution{DetailSource: "producer_locator", Custody: "producer_owned", CandidateCount: 1, ResolvedPath: selected}
		case filepath.Ext(selected) == ".jsonl":
			evidence := map[string]bool{"parent_session": true, "agent_type": true, "agent_id_prefix": true, "assignment": true, "launch_time": true}
			return childDetailResolution{DetailSource: "owned_session_fallback", Custody: "qualified_unique_join", JoinEvidence: evidence, CandidateCount: 1, ResolvedPath: selected}
		}
	}
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

func selectedLedgerSourcePath(session string, events []ledgerEvent) string {
	parent := canonicalPath(session)
	fallback := ""
	for _, event := range events {
		source, _ := event["source"].(map[string]any)
		path, _ := source["path"].(string)
		if path == "" {
			continue
		}
		if fallback == "" {
			fallback = path
		}
		if canonicalPath(path) != parent {
			return path
		}
	}
	return fallback
}
