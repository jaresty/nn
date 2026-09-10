package cmd

import (
	"sort"
)

// Keep acquisition local to one worker. The parent is shared for canonical
// identity, lifecycle and complete assignment joins; no all-worker raw capture
// is needed for the already-retained attention projections.
func observeParentCapture(session, canonical string) (*transcriptCapture, error) {
	source, err := captureTranscriptSource(canonical)
	if err != nil {
		return nil, err
	}
	c := &transcriptCapture{InputPath: session, Path: canonical, Sources: map[string]capturedTranscriptSource{canonical: source}, AuthPaths: map[string]string{}}
	c.indexLedgers()
	return c, nil
}

// Roster selection needs parent topology, not hydrated worker usage.
func observeRoster(session string) (treeChildPage, *transcriptCapture, error) {
	canonical, err := contextPath(session)
	if err != nil {
		return treeChildPage{}, nil, err
	}
	var parent *transcriptCapture
	var agents []agent
	if classifyTranscript(canonical) == schemaPi {
		parent, err = observeParentCapture(session, canonical)
		if err == nil {
			agents, err = buildPiTreeUsing(canonical, parent.read, parent.resolve)
		}
	} else {
		agents, err = buildTree(session)
	}
	if err != nil {
		return treeChildPage{}, nil, err
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i].ID < agents[j].ID })
	page, err := buildTreeChildPage(agents, "ROOT", 2, "")
	return page, parent, err
}

// Bounding the detector input must preserve its exact work-message suffix,
// including intervening non-work records and candidate-size validation.
func observeWorkWindow(records []ledgerRecord, n int) ([]ledgerRecord, int) {
	count := 0
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].Lifecycle {
			continue
		}
		_, role := attentionRecordRole(records[i])
		if role == "assistant" || role == "toolResult" || role == "tool" || role == "unknown" {
			count++
			if count == n {
				return records[i:], i
			}
		}
	}
	return records, 0
}
func sortObserveRecords(records []ledgerRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].Path != records[j].Path {
			return records[i].Path < records[j].Path
		}
		return records[i].Record.RecordOrdinal < records[j].Record.RecordOrdinal
	})
}
