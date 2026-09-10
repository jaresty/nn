package cmd

import "time"

const observeUnknownHistoryLimit = 20

// This is a source-change selection clock, not a claim about event timestamps,
// activity or health. Parent file mtime is deliberately not an input.
func observeWorkerRecency(source observeSourceStamp, err error, parentLatest time.Time, parentUnknown bool, cutoff time.Time) string {
	if !parentLatest.IsZero() && !parentLatest.Before(cutoff) {
		return "recent"
	}
	if err != nil || !source.Stable {
		return "unknown"
	}
	if !time.Unix(0, source.Modified).Before(cutoff) {
		return "recent"
	}
	if parentUnknown {
		return "unknown"
	}
	return "old"
}
