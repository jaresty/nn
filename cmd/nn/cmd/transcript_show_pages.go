package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"unicode/utf8"
)

type transcriptShowSegment struct {
	Segment  int    `json:"segment"`
	Segments int    `json:"segments"`
	Text     string `json:"text"`
}

type transcriptShowPage struct {
	Snapshot string                  `json:"snapshot"`
	Page     int                     `json:"page"`
	Pages    int                     `json:"pages"`
	NextPage int                     `json:"next_page"`
	Mode     string                  `json:"mode"`
	Segments []transcriptShowSegment `json:"segments"`
}

type transcriptShowSnapshotRequest struct {
	Version int    `json:"version"`
	Session string `json:"session"`
	AgentID string `json:"agent_id"`
	Mode    string `json:"mode"`
}

func buildTranscriptShowPage(session, agentID string, raw bool, text string, page int, suppliedSnapshot string) (transcriptShowPage, error) {
	if page < 1 {
		return transcriptShowPage{}, fmt.Errorf("transcript show: --page must be at least 1")
	}
	if page > 1 && suppliedSnapshot == "" {
		return transcriptShowPage{}, fmt.Errorf("transcript show: --snapshot is required for page %d", page)
	}
	if !utf8.ValidString(text) {
		return transcriptShowPage{}, fmt.Errorf("transcript show: projected output is not valid UTF-8")
	}
	mode := "meaningful"
	if raw {
		mode = "raw"
	}
	snapshot, err := transcriptShowSnapshot(session, agentID, mode, text)
	if err != nil {
		return transcriptShowPage{}, err
	}
	if suppliedSnapshot != "" && suppliedSnapshot != snapshot {
		return transcriptShowPage{}, fmt.Errorf("transcript show: stale or mismatched --snapshot; transcript, agent, or mode changed")
	}
	chunks := splitGraphBody(text)
	segments := make([]transcriptShowSegment, len(chunks))
	for i, chunk := range chunks {
		segments[i] = transcriptShowSegment{Segment: i + 1, Segments: len(chunks), Text: chunk}
	}
	pages, err := packTranscriptShowSegments(snapshot, mode, segments)
	if err != nil {
		return transcriptShowPage{}, err
	}
	if page > len(pages) {
		return transcriptShowPage{}, fmt.Errorf("transcript show: --page %d out of range (pages: %d)", page, len(pages))
	}
	result := transcriptShowPage{Snapshot: snapshot, Page: page, Pages: len(pages), Mode: mode, Segments: pages[page-1]}
	if page < len(pages) {
		result.NextPage = page + 1
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return transcriptShowPage{}, err
	}
	if len(encoded)+1 > graphBodiesPageMaxBytes {
		return transcriptShowPage{}, fmt.Errorf("transcript show: encoded page is %d bytes (limit %d)", len(encoded)+1, graphBodiesPageMaxBytes)
	}
	return result, nil
}

func transcriptShowSnapshot(session, agentID, mode, text string) (string, error) {
	absolute, err := filepath.Abs(session)
	if err != nil {
		return "", fmt.Errorf("transcript show: snapshot path: %w", err)
	}
	request, err := json.Marshal(transcriptShowSnapshotRequest{Version: 1, Session: filepath.Clean(absolute), AgentID: agentID, Mode: mode})
	if err != nil {
		return "", err
	}
	h := sha256.New()
	writeSnapshotPart(h, []byte("nn transcript show snapshot v1\x00"))
	writeSnapshotPart(h, request)
	writeSnapshotPart(h, []byte(text))
	return hex.EncodeToString(h.Sum(nil)), nil
}

func packTranscriptShowSegments(snapshot, mode string, segments []transcriptShowSegment) ([][]transcriptShowSegment, error) {
	pages := [][]transcriptShowSegment{{}}
	for _, segment := range segments {
		last := len(pages) - 1
		candidate := append(append([]transcriptShowSegment(nil), pages[last]...), segment)
		fits, err := transcriptShowSegmentsFit(snapshot, mode, candidate)
		if err != nil {
			return nil, err
		}
		if fits {
			pages[last] = candidate
			continue
		}
		fits, err = transcriptShowSegmentsFit(snapshot, mode, []transcriptShowSegment{segment})
		if err != nil {
			return nil, err
		}
		if !fits {
			return nil, fmt.Errorf("transcript show: segment %d cannot fit in one page", segment.Segment)
		}
		pages = append(pages, []transcriptShowSegment{segment})
	}
	return pages, nil
}

func transcriptShowSegmentsFit(snapshot, mode string, segments []transcriptShowSegment) (bool, error) {
	candidate := transcriptShowPage{
		Snapshot: snapshot, Page: math.MaxInt64, Pages: math.MaxInt64,
		NextPage: math.MaxInt64, Mode: mode, Segments: segments,
	}
	encoded, err := json.Marshal(candidate)
	if err != nil {
		return false, err
	}
	return len(encoded)+1 <= graphBodiesPageMaxBytes, nil
}
