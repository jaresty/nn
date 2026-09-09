package cmd

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Each source is an independently captured complete-record prefix. This is not
// a simultaneous filesystem snapshot. Append-only producers may keep writing.
type capturedTranscriptSource struct {
	PrefixBytes int64       `json:"prefix_bytes"`
	Digest      string      `json:"sha256"`
	Records     []rawRecord `json:"-"`
	Raw         []byte      `json:"bytes"`
}
type transcriptCapture struct {
	Ledgers   map[string][]ledgerRecord           `json:"-"`
	Details   map[string]string                   `json:"-"`
	InputPath string                              `json:"input_path"`
	ID        string                              `json:"-"`
	Path      string                              `json:"path"`
	Sources   map[string]capturedTranscriptSource `json:"sources"`
	AuthPaths map[string]string                   `json:"authenticated_paths"`
}
type captureBinding struct {
	Capture, Request string
	Pages            []string
}

func captureCacheDir() (string, error) {
	base, e := os.UserCacheDir()
	if e != nil {
		return "", e
	}
	dir := filepath.Join(base, "nn", "transcript-captures-v1")
	if e = os.MkdirAll(dir, 0700); e != nil {
		return "", e
	}
	info, e := os.Lstat(dir)
	if e != nil {
		return "", e
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("unsafe capture cache directory")
	}
	if e = os.Chmod(dir, 0700); e != nil {
		return "", e
	}
	return dir, nil
}
func captureHash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func validCaptureID(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 32 && strings.ToLower(s) == s
}
func writeCaptureFile(name string, b []byte) error {
	dir, e := captureCacheDir()
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(dir, ".capture-")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), filepath.Join(dir, name))
}
func readCaptureFile(name string) ([]byte, error) {
	dir, e := captureCacheDir()
	if e != nil {
		return nil, e
	}
	path := filepath.Join(dir, name)
	info, e := os.Lstat(path)
	if e != nil {
		return nil, fmt.Errorf("capture unavailable; explicitly refresh: %w", e)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("capture has unsafe file permissions or type")
	}
	if time.Since(info.ModTime()) > 24*time.Hour {
		return nil, fmt.Errorf("capture expired; explicitly refresh")
	}
	return os.ReadFile(path)
}

func captureTranscriptSource(path string) (capturedTranscriptSource, error) {
	var source capturedTranscriptSource
	f, e := os.Open(path)
	if e != nil {
		return source, e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil {
		return source, e
	}
	if !info.Mode().IsRegular() {
		return source, fmt.Errorf("capture source must be regular")
	}
	b, e := io.ReadAll(io.LimitReader(f, info.Size()))
	if e != nil {
		return source, e
	}
	if int64(len(b)) != info.Size() {
		return source, fmt.Errorf("capture source truncated during prefix read")
	}
	// Exclude an unfinished final record, even when the writer is mid-JSON.
	end := bytes.LastIndexByte(b, '\n') + 1
	b = b[:end]
	source.PrefixBytes = int64(end)
	source.Digest = captureHash(b)
	source.Raw = b
	source.Records, e = parseCapturedRecords(b)
	return source, e
}

func parseCapturedRecords(b []byte) ([]rawRecord, error) {
	records := []rawRecord{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 65536), 16*1024*1024)
	for sc.Scan() {
		var r rawRecord
		if json.Unmarshal(sc.Bytes(), &r) == nil {
			r.RecordOrdinal = len(records) + 1
			records = append(records, r)
		}
	}
	return records, sc.Err()
}

func newTranscriptCapture(session string) (*transcriptCapture, error) {
	cleanupTranscriptCaptures()
	path, e := contextPath(session)
	if e != nil {
		return nil, e
	}
	root, e := captureTranscriptSource(path)
	if e != nil {
		return nil, e
	}
	pi := false
	for _, r := range root.Records {
		if r.Type == "session" || r.CustomType == "subagents:record" {
			pi = true
			break
		}
	}
	if !pi {
		return nil, fmt.Errorf("capture requires a Pi transcript")
	}
	input, _ := filepath.Abs(session)
	c := &transcriptCapture{InputPath: input, Path: path, Sources: map[string]capturedTranscriptSource{path: root}, AuthPaths: map[string]string{}}
	for _, loc := range piBackgroundLocators(root.Records) {
		safe := validatePiSidechainPath(loc.Path, loc.AgentID)
		if safe == "" {
			continue
		}
		if _, ok := c.Sources[safe]; !ok {
			source, e := captureTranscriptSource(safe)
			if e != nil {
				continue
			}
			c.Sources[safe] = source
		}
		c.AuthPaths[loc.Path+"\x00"+loc.AgentID] = safe
	}
	b, e := json.Marshal(c)
	if e != nil {
		return nil, e
	}
	c.ID = captureHash(b)
	if e = writeCaptureFile(c.ID+".json", b); e != nil {
		return nil, e
	}
	c.indexLedgers()
	return c, nil
}
func loadTranscriptCapture(id string) (*transcriptCapture, error) {
	if !validCaptureID(id) {
		return nil, fmt.Errorf("invalid capture ID")
	}
	b, e := readCaptureFile(id + ".json")
	if e != nil {
		return nil, e
	}
	if captureHash(b) != id {
		return nil, fmt.Errorf("capture digest mismatch")
	}
	var c transcriptCapture
	if e = json.Unmarshal(b, &c); e != nil {
		return nil, e
	}
	c.ID = id
	for path, s := range c.Sources {
		if int64(len(s.Raw)) != s.PrefixBytes || captureHash(s.Raw) != s.Digest {
			return nil, fmt.Errorf("capture source digest mismatch")
		}
		s.Records, e = parseCapturedRecords(s.Raw)
		if e != nil {
			return nil, e
		}
		c.Sources[path] = s
	}
	c.indexLedgers()
	return &c, nil
}
func (c *transcriptCapture) read(path string) ([]rawRecord, error) {
	s, ok := c.Sources[path]
	if !ok {
		return nil, fmt.Errorf("source unavailable in capture")
	}
	return s.Records, nil
}
func (c *transcriptCapture) resolve(path, id string) string { return c.AuthPaths[path+"\x00"+id] }
func (c *transcriptCapture) digests() map[string]string {
	out := map[string]string{}
	for p, s := range c.Sources {
		out[p] = s.Digest
	}
	return out
}
func (c *transcriptCapture) prefixes() map[string]int64 {
	out := map[string]int64{}
	for p, s := range c.Sources {
		out[p] = s.PrefixBytes
	}
	return out
}
func (c *transcriptCapture) indexLedgers() {
	c.Ledgers = map[string][]ledgerRecord{}
	c.Details = map[string]string{}
	root := c.Sources[c.Path].Records
	owned := map[string][]rawRecord{}
	lifecycle := map[string][]ledgerRecord{}
	for _, r := range root {
		if isPiEventRecord(r) {
			id := r.AgentID
			if id == "" {
				id = "ROOT"
			}
			owned[id] = append(owned[id], r)
		}
		if r.Type == "custom" && r.CustomType == "subagents:record" {
			var d piCustomData
			if json.Unmarshal(r.Data, &d) == nil && d.ID != "" {
				lifecycle[d.ID] = append(lifecycle[d.ID], ledgerRecord{r, c.Path, true})
			}
		}
	}
	paths := map[string]string{}
	for id := range owned {
		paths[id] = c.Path
	}
	seen := map[string]bool{}
	for _, loc := range piBackgroundLocators(root) {
		if seen[loc.AgentID] || len(owned[loc.AgentID]) > 0 {
			continue
		}
		seen[loc.AgentID] = true
		source := c.resolve(loc.Path, loc.AgentID)
		owned[loc.AgentID] = ownedPiRecords(c.Sources[source].Records, loc.AgentID, true)
		paths[loc.AgentID] = source
	}
	for id, rs := range lifecycle {
		c.Ledgers[id] = append([]ledgerRecord{}, rs...)
	}
	for id, rs := range owned {
		for _, r := range rs {
			c.Ledgers[id] = append(c.Ledgers[id], ledgerRecord{r, paths[id], false})
		}
		for _, r := range rs {
			var m map[string]json.RawMessage
			if json.Unmarshal(r.Message, &m) == nil && m != nil {
				c.Details[id] = "available"
				break
			}
		}
	}
	for id, rs := range c.Ledgers {
		sort.SliceStable(rs, func(i, j int) bool {
			if rs[i].Path != rs[j].Path {
				return rs[i].Path < rs[j].Path
			}
			return rs[i].Record.RecordOrdinal < rs[j].Record.RecordOrdinal
		})
		c.Ledgers[id] = rs
	}
}
func (c *transcriptCapture) ledger(id string) ([]ledgerRecord, string) {
	status := c.Details[id]
	if status == "" {
		status = "unavailable"
	}
	return c.Ledgers[id], status
}
func captureRequest(session string, options ...any) (string, error) {
	p, e := filepath.Abs(session)
	if e != nil {
		return "", e
	}
	b, e := json.Marshal(append([]any{filepath.Clean(p)}, options...))
	return captureHash(b), e
}
func saveCapturedPages(first ledgerPage, request string, c *transcriptCapture, build func(int) (ledgerPage, error)) error {
	binding := captureBinding{Capture: c.ID, Request: request}
	for n := 1; n <= first.Pages; n++ {
		p := first
		var e error
		if n != first.Page {
			p, e = build(n)
			if e != nil {
				return e
			}
		}
		b, e := json.Marshal(p)
		if e != nil {
			return e
		}
		digest := captureHash(b)
		if e = writeCaptureFile(digest+".page", b); e != nil {
			return e
		}
		binding.Pages = append(binding.Pages, digest)
	}
	b, e := json.Marshal(binding)
	if e != nil {
		return e
	}
	return writeCaptureFile(first.Snapshot+".binding", b)
}

func loadCapturedPage(snapshot, request string, page int) (ledgerPage, error) {
	var result ledgerPage
	if !validCaptureID(snapshot) {
		return result, fmt.Errorf("invalid bundle snapshot")
	}
	b, e := readCaptureFile(snapshot + ".binding")
	if e != nil {
		return result, e
	}
	var binding captureBinding
	if json.Unmarshal(b, &binding) != nil || binding.Request != request {
		return result, fmt.Errorf("capture request mismatch")
	}
	if page < 1 || page > len(binding.Pages) {
		return result, fmt.Errorf("capture page out of range")
	}
	digest := binding.Pages[page-1]
	if !validCaptureID(digest) {
		return result, fmt.Errorf("invalid captured page ID")
	}
	b, e = readCaptureFile(digest + ".page")
	if e != nil {
		return result, e
	}
	if captureHash(b) != digest {
		return result, fmt.Errorf("captured page digest mismatch")
	}
	if e = json.Unmarshal(b, &result); e != nil {
		return result, e
	}
	if result.Snapshot != snapshot || result.Page != page {
		return ledgerPage{}, fmt.Errorf("captured page identity mismatch")
	}
	return result, nil
}

// Expiry bounds retained sensitive transcript data; expired captures never refresh implicitly.
func cleanupTranscriptCaptures() {
	dir, e := captureCacheDir()
	if e != nil {
		return
	}
	entries, e := os.ReadDir(dir)
	if e != nil {
		return
	}
	for _, entry := range entries {
		if !(strings.HasSuffix(entry.Name(), ".json") || strings.HasSuffix(entry.Name(), ".binding") || strings.HasSuffix(entry.Name(), ".page")) {
			continue
		}
		info, e := entry.Info()
		if e == nil && info.Mode().IsRegular() && time.Since(info.ModTime()) > 24*time.Hour {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}
