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
