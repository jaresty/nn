package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
)

type summaryComponents struct {
	Input         *int64 `json:"input_tokens"`
	Output        *int64 `json:"output_tokens"`
	CacheRead     *int64 `json:"cache_read_tokens"`
	CacheCreation *int64 `json:"cache_creation_tokens"`
}
type summaryUsageRecord struct {
	ID         string
	Components summaryComponents
}
type summaryContext struct {
	Known   int      `json:"known_records"`
	Unknown int      `json:"unknown_records"`
	Zero    int      `json:"zero_records"`
	First   *int64   `json:"first"`
	Last    *int64   `json:"last"`
	Min     *int64   `json:"min"`
	Max     *int64   `json:"max"`
	Average *float64 `json:"average_known"`
}
type summaryUsageStats struct {
	Records     int               `json:"records"`
	Complete    int               `json:"complete_records"`
	Partial     int               `json:"partial_records"`
	Unavailable int               `json:"unavailable_records"`
	Zero        int               `json:"zero_usage_records"`
	Status      string            `json:"status"`
	Totals      summaryComponents `json:"totals"`
	Missing     map[string]int    `json:"missing_counts"`
	KnownTotal  int64             `json:"known_total_tokens"`
	Total       *int64            `json:"total_tokens"`
	Context     summaryContext    `json:"context"`
}
type summaryUsageBucket struct {
	First   int               `json:"first_record"`
	Last    int               `json:"last_record"`
	FirstID string            `json:"first_event_id"`
	LastID  string            `json:"last_event_id"`
	Stats   summaryUsageStats `json:"stats"`
}

func aggregateSummaryUsage(rows []summaryUsageRecord) (summaryUsageStats, error) {
	names := []string{"input_tokens", "output_tokens", "cache_read_tokens", "cache_creation_tokens"}
	s := summaryUsageStats{Records: len(rows), Status: "unavailable", Missing: map[string]int{}}
	for _, name := range names {
		s.Missing[name] = 0
	}
	destinations := []**int64{&s.Totals.Input, &s.Totals.Output, &s.Totals.CacheRead, &s.Totals.CacheCreation}
	var contextSum int64
	for i, r := range rows {
		values := []*int64{r.Components.Input, r.Components.Output, r.Components.CacheRead, r.Components.CacheCreation}
		known := 0
		var recordTotal int64
		for j, value := range values {
			if value == nil {
				s.Missing[names[j]]++
				continue
			}
			if *value < 0 || *value > math.MaxInt64-s.KnownTotal {
				return s, fmt.Errorf("usage summary: negative counter or integer overflow")
			}
			s.KnownTotal += *value
			recordTotal += *value
			known++
			if *destinations[j] == nil {
				*destinations[j] = new(int64)
			}
			**destinations[j] += *value
		}
		switch known {
		case 0:
			s.Unavailable++
		case 4:
			s.Complete++
			if recordTotal == 0 {
				s.Zero++
			}
		default:
			s.Partial++
		}
		var context *int64
		if r.Components.Input != nil && r.Components.CacheRead != nil {
			value := *r.Components.Input + *r.Components.CacheRead
			context = &value
			s.Context.Known++
			contextSum += value
			if value == 0 {
				s.Context.Zero++
			}
			if s.Context.Min == nil || value < *s.Context.Min {
				s.Context.Min = context
			}
			if s.Context.Max == nil || value > *s.Context.Max {
				s.Context.Max = context
			}
		} else {
			s.Context.Unknown++
		}
		if i == 0 {
			s.Context.First = context
		}
		s.Context.Last = context
	}
	if s.Complete > 0 || s.Partial > 0 {
		s.Status = "partial"
	}
	if len(rows) > 0 && s.Complete == len(rows) {
		s.Status = "complete"
		value := s.KnownTotal
		s.Total = &value
	}
	if s.Context.Known > 0 {
		mean := float64(contextSum) / float64(s.Context.Known)
		s.Context.Average = &mean
	}
	return s, nil
}

func summarizeLedgerUsage(events []ledgerEvent, bucketSize int) (map[string]any, error) {
	if bucketSize < 0 {
		return nil, fmt.Errorf("usage summary: --bucket-size must not be negative")
	}
	rows := []summaryUsageRecord{}
	for _, e := range events {
		if e["kind"] != "message" || e["usage"] == nil {
			continue
		}
		b, err := json.Marshal(e["usage"])
		if err != nil {
			return nil, err
		}
		var components summaryComponents
		if err = json.Unmarshal(b, &components); err != nil {
			return nil, err
		}
		id, _ := e["event_id"].(string)
		rows = append(rows, summaryUsageRecord{id, components})
	}
	stats, err := aggregateSummaryUsage(rows)
	if err != nil {
		return nil, err
	}
	buckets := []summaryUsageBucket{}
	if bucketSize > 0 {
		for start := 0; start < len(rows); {
			end := len(rows)
			if bucketSize < len(rows)-start {
				end = start + bucketSize
			}
			part, err := aggregateSummaryUsage(rows[start:end])
			if err != nil {
				return nil, err
			}
			buckets = append(buckets, summaryUsageBucket{start + 1, end, rows[start].ID, rows[end-1].ID, part})
			start = end
		}
	}
	return map[string]any{"stats": stats, "buckets": buckets}, nil
}

func buildUsageSummary(session, id, schema, detail string, events []ledgerEvent, bucketSize int, supplied string) ([]byte, error) {
	result, err := summarizeLedgerUsage(events, bucketSize)
	if err != nil {
		return nil, err
	}
	// Reuse the ledger snapshot contract without packing pages or returning event payloads.
	ledger, err := buildLedgerPage(session, id, schema, detail, []string{"identity", "usage"}, false, events, 1, "", "", true)
	if err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(session)
	if err != nil {
		return nil, err
	}
	result["session"] = filepath.Clean(absolute)
	result["agent_id"] = id
	result["version"] = "nn.transcript.usage-summary/v1"
	result["ledger_snapshot"] = ledger.Snapshot
	result["schema"] = schema
	result["detail_status"] = detail
	result["bucket_size"] = bucketSize
	body, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(append([]byte("nn transcript usage summary v1\x00"), body...))
	snapshot := hex.EncodeToString(h[:])
	if supplied != "" && supplied != snapshot {
		return nil, fmt.Errorf("usage summary: stale or mismatched --snapshot")
	}
	result["snapshot"] = snapshot
	body, err = json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if len(body)+1 > graphBodiesPageMaxBytes {
		return nil, fmt.Errorf("usage summary exceeds 48000 bytes; increase --bucket-size or omit buckets")
	}
	return append(body, '\n'), nil
}
