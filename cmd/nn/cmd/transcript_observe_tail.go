package cmd

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type observeTailRecord struct {
	Ordinal int       `json:"ordinal"`
	Record  rawRecord `json:"record"`
	Index   int       `json:"owned_index"`
}
type observeTail struct {
	Version         int                 `json:"version"`
	Path            string              `json:"path"`
	Agent           string              `json:"agent"`
	Limit           int                 `json:"limit"`
	Stamp           observeSourceStamp  `json:"stamp"`
	Offset          int64               `json:"offset"`
	Ordinal         int                 `json:"ordinal"`
	Owned           int                 `json:"owned"`
	Available       bool                `json:"available"`
	Latest          time.Time           `json:"latest"`
	Unknown         bool                `json:"unknown"`
	Prefix          string              `json:"prefix"`
	OwnedHash       []byte              `json:"owned_hash_state"`
	Work            []observeTailRecord `json:"work"`
	Users           []observeTailRecord `json:"users"`
	TaskTotal       int                 `json:"task_total"`
	TaskUnavailable int                 `json:"task_unavailable"`
	Oversized       *observeTailRecord  `json:"oversized,omitempty"`
}

func retainObserveTail(s observeTail) (string, error) {
	// Decline clearly oversized caches before allocating their encoded copy.
	remaining := 4 * 1024 * 1024
	rows := [][]observeTailRecord{s.Work, s.Users}
	if s.Oversized != nil {
		rows = append(rows, []observeTailRecord{*s.Oversized})
	}
	for _, group := range rows {
		for _, row := range group {
			remaining -= len(row.Record.Message) + len(row.Record.Data)
			if remaining < 0 {
				return "", nil
			}
		}
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	if len(b) > 4*1024*1024 {
		return "", nil
	} // caching is optional, evidence is not
	id := captureHash(b)
	return id, writeCaptureFile(id+".observe-tail", b)
}
func loadObserveTail(id, path, agent string, n int) *observeTail {
	if !validCaptureID(id) {
		return nil
	}
	b, err := readCaptureFile(id + ".observe-tail")
	if err != nil || captureHash(b) != id {
		return nil
	}
	var s observeTail
	if json.Unmarshal(b, &s) != nil || s.Version != 2 || s.Path != path || s.Agent != agent || s.Limit != n || s.Offset < 0 {
		return nil
	}
	for _, rows := range [][]observeTailRecord{s.Work, s.Users} {
		for i := range rows {
			if rows[i].Ordinal < 1 {
				return nil
			}
			rows[i].Record.RecordOrdinal = rows[i].Ordinal
		}
	}
	if s.Oversized != nil {
		if s.Oversized.Ordinal < 1 {
			return nil
		}
		s.Oversized.Record.RecordOrdinal = s.Oversized.Ordinal
	}
	return &s
}
func appendObserveTail(rows []observeTailRecord, r observeTailRecord, n int) []observeTailRecord {
	if len(rows) == n {
		copy(rows, rows[1:])
		rows[len(rows)-1] = r
		return rows
	}
	return append(rows, r)
}

// Scan complete lines only. The byte offset includes malformed complete lines;
// ordinals count only decoder-accepted records, matching parseCapturedRecords.
func scanObserveTail(path, agent string, n int, prior *observeTail) (observeTail, bool, error) {
	var zero observeTail
	if n < 1 || n > 200 {
		return zero, false, fmt.Errorf("observe: invalid worker window")
	}
	f, err := os.Open(path)
	if err != nil {
		return zero, false, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return zero, false, err
	}
	if !info.Mode().IsRegular() {
		return zero, false, fmt.Errorf("observe: worker source must be regular")
	}
	stamp := stampObserveSource(path, info)
	s := observeTail{Version: 2, Path: path, Agent: agent, Limit: n, Stamp: stamp}
	prefix := sha256.New()
	owned := sha256.New()
	incremental := false
	if prior != nil && prior.Version == 2 && prior.Offset >= 0 && prior.Path == path && prior.Agent == agent && prior.Limit == n && prior.Stamp.Stable && stamp.Stable && prior.Stamp.Identity == stamp.Identity && prior.Offset <= info.Size() {
		if _, e := io.CopyN(prefix, f, prior.Offset); e == nil && hex.EncodeToString(prefix.Sum(nil)) == prior.Prefix {
			if e = owned.(encoding.BinaryUnmarshaler).UnmarshalBinary(prior.OwnedHash); e == nil {
				s = *prior
				s.Work = append([]observeTailRecord{}, prior.Work...)
				s.Users = append([]observeTailRecord{}, prior.Users...)
				s.Stamp = stamp
				incremental = true
			}
		}
	}
	if !incremental {
		if _, err = f.Seek(0, io.SeekStart); err != nil {
			return zero, false, err
		}
		prefix.Reset()
		owned.Reset()
	}
	sc := bufio.NewScanner(io.LimitReader(f, info.Size()-s.Offset))
	sc.Buffer(make([]byte, 65536), 16*1024*1024)
	sc.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			return i + 1, data[:i], nil
		}
		if atEOF {
			return len(data), nil, nil
		}
		return 0, nil, nil
	})
	decoded := uint64(0)
	defer func() { transcriptDecodeCount.Add(decoded) }()
	encoder := json.NewEncoder(owned)
	for sc.Scan() {
		line := sc.Bytes()
		prefix.Write(line)
		prefix.Write([]byte{'\n'})
		s.Offset += int64(len(line) + 1)
		var r rawRecord
		if json.Unmarshal(line, &r) != nil {
			continue
		}
		decoded++
		s.Ordinal++
		r.RecordOrdinal = s.Ordinal
		if !isPiEventRecord(r) || r.AgentID != agent {
			continue
		}
		if err = encoder.Encode(r); err != nil {
			return zero, incremental, err
		}
		row := observeTailRecord{Record: r, Index: s.Owned, Ordinal: r.RecordOrdinal}
		s.Owned++
		lr := ledgerRecord{Record: r, Path: path}
		m, role := attentionRecordRole(lr)
		if m != nil {
			s.Available = true
		}
		var value any = r.Timestamp
		if r.Timestamp == "" {
			value = nil
			_ = json.Unmarshal(m["timestamp"], &value)
		}
		tm, status := ledgerTime(value)
		if status != "known" {
			s.Unknown = true
		} else if tm.After(s.Latest) {
			s.Latest = tm
		}
		if role == "assistant" || role == "toolResult" || role == "tool" || role == "unknown" {
			s.Work = appendObserveTail(s.Work, row, n)
		}
		originalRole := ledgerString(m, "role")
		if originalRole == "user" || (originalRole == "" && r.Type == "user") {
			task := attentionTaskContextFor(agent, path, []ledgerRecord{lr}, nil)
			s.TaskTotal += task.Total
			s.TaskUnavailable += task.Unavailable
			if len(task.Evidence) > 0 {
				s.Users = appendObserveTail(s.Users, row, 2)
			}
		}
		if len(r.Message) > 1024*1024 {
			copy := row
			s.Oversized = &copy
		}
	}
	if err = sc.Err(); err != nil {
		return zero, incremental, err
	}
	s.Prefix = hex.EncodeToString(prefix.Sum(nil))
	s.OwnedHash, err = owned.(encoding.BinaryMarshaler).MarshalBinary()
	if err != nil {
		return zero, incremental, err
	}
	after, e := currentObserveStamp(path)
	if e != nil || !sameObserveStamp(stamp, after) {
		if incremental {
			return scanObserveTail(path, agent, n, nil)
		}
		s.Stamp.Stable = false
	}
	return s, incremental, nil
}
func (s observeTail) records() []ledgerRecord {
	byOrdinal := map[int]rawRecord{}
	for _, rows := range [][]observeTailRecord{s.Work, s.Users} {
		for _, r := range rows {
			byOrdinal[r.Record.RecordOrdinal] = r.Record
		}
	}
	if s.Oversized != nil {
		byOrdinal[s.Oversized.Record.RecordOrdinal] = s.Oversized.Record
	}
	out := make([]ledgerRecord, 0, len(byOrdinal))
	for _, r := range byOrdinal {
		out = append(out, ledgerRecord{Record: r, Path: s.Path})
	}
	sortObserveRecords(out)
	return out
}
func (s observeTail) earlier(event string, parent []ledgerRecord) (int, bool) {
	for _, row := range s.Work {
		if ledgerID(s.Path, row.Record.RecordOrdinal, s.Agent, "message") == event {
			count := row.Index
			for _, r := range parent {
				if r.Path < s.Path || (r.Path == s.Path && r.Record.RecordOrdinal < row.Record.RecordOrdinal) {
					count++
				}
			}
			return count, true
		}
	}
	return 0, false
}
func (s observeTail) digest() string {
	h := sha256.New()
	if h.(encoding.BinaryUnmarshaler).UnmarshalBinary(s.OwnedHash) != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}
