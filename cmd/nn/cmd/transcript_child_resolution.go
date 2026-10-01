package cmd

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type childDetailResolution struct {
	DetailSource                  string                     `json:"detail_source,omitempty"`
	Custody                       string                     `json:"custody,omitempty"`
	JoinEvidence                  map[string]bool            `json:"join_evidence,omitempty"`
	CandidateCount                int                        `json:"candidate_count,omitempty"`
	ResolvedPath                  string                     `json:"resolved_path,omitempty"`
	ResolutionCandidates          []childResolutionCandidate `json:"resolution_candidates,omitempty"`
	ResolutionCandidateTotal      int                        `json:"resolution_candidate_total,omitempty"`
	ResolutionCandidatesTruncated bool                       `json:"resolution_candidates_truncated,omitempty"`
}

type childResolutionCandidate struct {
	Session      string          `json:"session"`
	Path         string          `json:"path"`
	Timestamp    string          `json:"timestamp,omitempty"`
	Label        string          `json:"label"`
	Status       string          `json:"status"`
	Checks       map[string]bool `json:"checks"`
	FailedChecks []string        `json:"failed_checks"`
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
		if !strings.HasPrefix(identity.Name, prefix) || !strings.HasPrefix(id, strings.TrimPrefix(identity.Name, prefix)) || identity.Assignment != assignment || launchErr != nil {
			return nil
		}
		candidateTime, candidateErr := time.Parse(time.RFC3339Nano, identity.FirstTimestamp)
		if candidateErr != nil || candidateTime.Before(launchTime) {
			return nil
		}
		rows, readErr := readRecords(path)
		if readErr == nil {
			candidates = append(candidates, piOwnedSessionCandidate{Path: canonicalPath(path), Records: ownedPiRecords(rows, "ROOT", false)})
		}
		return nil
	})
	if len(candidates) > 0 {
		for key := range evidence {
			evidence[key] = true
		}
	}
	return candidates, evidence
}

func piOwnedSessionScan(parent string, recs []rawRecord, id string) ([]piOwnedSessionCandidate, map[string]bool, []childResolutionCandidate) {
	_, agentType, assignment, launched, ok := piLaunchForChild(recs, id)
	evidence := map[string]bool{"parent_session": false, "agent_type": false, "agent_id_prefix": false, "assignment": false, "launch_time": false}
	if !ok {
		return nil, evidence, nil
	}
	parent = canonicalPath(parent)
	launchTime, launchErr := time.Parse(time.RFC3339Nano, launched)
	var qualified []piOwnedSessionCandidate
	var diagnostics []childResolutionCandidate
	_ = filepath.WalkDir(filepath.Dir(parent), func(path string, entry fs.DirEntry, err error) error {
		canonical := canonicalPath(path)
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".jsonl" || canonical == parent {
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
		checks := map[string]bool{"parent_session": canonicalPath(recorded) == parent, "agent_type": false, "agent_id_prefix": false, "assignment": false, "launch_time": false}
		if !checks["parent_session"] {
			return nil
		}
		prefix := agentType + "#"
		checks["agent_type"] = strings.HasPrefix(identity.Name, prefix)
		candidateID := strings.TrimPrefix(identity.Name, prefix)
		checks["agent_id_prefix"] = checks["agent_type"] && strings.HasPrefix(id, candidateID)
		checks["assignment"] = identity.Assignment == assignment
		if launchErr == nil {
			candidateTime, candidateErr := time.Parse(time.RFC3339Nano, identity.FirstTimestamp)
			checks["launch_time"] = candidateErr == nil && !candidateTime.Before(launchTime)
		}
		failed := make([]string, 0, len(checks))
		for _, key := range []string{"agent_type", "agent_id_prefix", "assignment", "launch_time"} {
			if !checks[key] {
				failed = append(failed, key)
			}
		}
		status := "unqualified"
		if len(failed) == 0 {
			status = "qualified"
			rows, readErr := readRecords(path)
			if readErr == nil {
				qualified = append(qualified, piOwnedSessionCandidate{Path: canonical, Records: ownedPiRecords(rows, "ROOT", false)})
			}
		}
		label, _ := transcriptDisplayLabel(identity.Assignment)
		diagnostics = append(diagnostics, childResolutionCandidate{Session: strings.TrimSuffix(filepath.Base(path), ".jsonl"), Path: canonical, Timestamp: identity.FirstTimestamp, Label: label, Status: status, Checks: checks, FailedChecks: failed})
		return nil
	})
	if len(qualified) > 0 {
		for key := range evidence {
			evidence[key] = true
		}
	}
	return qualified, evidence, diagnostics
}

func boundResolutionCandidates(candidates []childResolutionCandidate) ([]childResolutionCandidate, int, bool) {
	total := len(candidates)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Timestamp != candidates[j].Timestamp {
			return candidates[i].Timestamp > candidates[j].Timestamp
		}
		return candidates[i].Session < candidates[j].Session
	})
	const limit = 10
	if len(candidates) > limit {
		return candidates[:limit], total, true
	}
	return candidates, total, false
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

func transcriptChildDetailResolutionWithCandidates(path, id, schema, detail string, selectedPaths ...string) childDetailResolution {
	resolution := transcriptChildDetailResolution(path, id, schema, detail, selectedPaths...)
	if schema != schemaPi || id == "" || id == "ROOT" {
		return resolution
	}
	canonical := canonicalPath(path)
	recs, err := readRecords(canonical)
	if err != nil {
		return resolution
	}
	_, _, diagnostics := piOwnedSessionScan(canonical, recs, id)
	resolution.ResolutionCandidates, resolution.ResolutionCandidateTotal, resolution.ResolutionCandidatesTruncated = boundResolutionCandidates(diagnostics)
	return resolution
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
