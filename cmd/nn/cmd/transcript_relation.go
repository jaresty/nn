package cmd

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type transcriptRelation struct {
	ConversationKind string
	OwnerSession     *string
	Authority        string
}

type transcriptRelationExtractor func(string) (transcriptRelation, bool)

var transcriptRelationExtractors = map[string]transcriptRelationExtractor{
	schemaPi: piTranscriptRelation,
}

func relationForTranscript(path string) transcriptRelation {
	if extract := transcriptRelationExtractors[classifyTranscript(path)]; extract != nil {
		if relation, ok := extract(path); ok {
			return relation
		}
	}
	return transcriptRelation{ConversationKind: "conversation", Authority: "unavailable"}
}

type piSessionHeader struct {
	Type          string `json:"type"`
	ID            string `json:"id"`
	ParentSession string `json:"parentSession"`
}

func piTranscriptRelation(path string) (transcriptRelation, bool) {
	header, ok := readPiSessionHeader(path)
	if !ok {
		return transcriptRelation{}, false
	}
	if header.ParentSession != "" {
		if owner, valid := validatedPiParent(path, header.ParentSession); valid {
			return transcriptRelation{ConversationKind: "sidechain", OwnerSession: &owner, Authority: "recorded"}, true
		}
	}
	if hasPiAgentExecutionAncestor(path) {
		return transcriptRelation{ConversationKind: "sidechain", Authority: "heuristic"}, true
	}
	return transcriptRelation{ConversationKind: "conversation", Authority: "unavailable"}, true
}

func readPiSessionHeader(path string) (piSessionHeader, bool) {
	f, err := os.Open(path)
	if err != nil {
		return piSessionHeader{}, false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var header piSessionHeader
		if json.Unmarshal([]byte(line), &header) != nil || header.Type != "session" || header.ID == "" {
			return piSessionHeader{}, false
		}
		return header, true
	}
	return piSessionHeader{}, false
}

func validatedPiParent(childPath, recordedPath string) (string, bool) {
	parentPath := recordedPath
	if !filepath.IsAbs(parentPath) {
		parentPath = filepath.Join(filepath.Dir(childPath), parentPath)
	}
	childAbs, childErr := filepath.Abs(childPath)
	parentAbs, parentErr := filepath.Abs(parentPath)
	if childErr != nil || parentErr != nil || filepath.Clean(childAbs) == filepath.Clean(parentAbs) {
		return "", false
	}
	parent, ok := readPiSessionHeader(parentAbs)
	if !ok {
		return "", false
	}
	return parent.ID, true
}

func hasPiAgentExecutionAncestor(path string) bool {
	for dir := filepath.Dir(path); dir != "." && dir != string(filepath.Separator); dir = filepath.Dir(dir) {
		if strings.Contains(filepath.Base(dir), "pi-agent-") {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return false
}
