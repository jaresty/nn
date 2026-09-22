package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestTranscriptArtifactReadUnsupportedPlatformError(t *testing.T) {
	if got := classifyArtifactOpenError(errArtifactUnsupported); got != errArtifactUnsupported {
		t.Fatalf("ASSERT_ARTIFACT_UNSUPPORTED_CLASSIFICATION: got %v", got)
	}
}

func TestTranscriptArtifactReadMissingRootAndReferenceBeforeOpener(t *testing.T) {
	for _, tc := range []struct{ name, path, root string }{
		{"missing-root", "/safe/result.json", ""},
		{"missing-reference", "", "/safe"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			records := artifactReadTestRecords(tc.path)
			event := artifactReadEventID(t, records)
			calls := 0
			c := newTranscriptArtifactReadCmdUsing(func(string, string) ([]ledgerRecord, string, string, error) { return records, "pi", "available", nil }, func(string, string) (artifactReadFile, error) { calls++; return nil, os.ErrPermission })
			c.SetArgs([]string{"session", "agent", "--event", event, "--snapshot", artifactReadTestSnapshot(t, records, event), "--finding", "1", "--allow-root", tc.root})
			if err := c.Execute(); err == nil || calls != 0 {
				t.Fatalf("ASSERT_ARTIFACT_INVALID_BEFORE_OPENER: case=%s calls=%d err=%v", tc.name, calls, err)
			}
		})
	}
}

func TestTranscriptArtifactReadStaleSnapshotBeforeOpener(t *testing.T) {
	calls := 0
	records := artifactReadTestRecords("/safe/result.json")
	event := artifactReadEventID(t, records)
	c := newTranscriptArtifactReadCmdUsing(func(string, string) ([]ledgerRecord, string, string, error) { return records, "pi", "available", nil }, func(string, string) (artifactReadFile, error) { calls++; return nil, os.ErrPermission })
	c.SetArgs([]string{"session", "agent", "--event", event, "--snapshot", strings.Repeat("0", 64), "--finding", "1", "--allow-root", "/safe"})
	err := c.Execute()
	if err == nil || !strings.Contains(err.Error(), "snapshot") {
		t.Fatalf("ASSERT_ARTIFACT_STALE_SNAPSHOT: %v", err)
	}
	if calls != 0 {
		t.Fatalf("ASSERT_ARTIFACT_STALE_SNAPSHOT_BEFORE_OPENER: calls=%d", calls)
	}
}

func TestTranscriptArtifactReadRejectedPathBeforeOpener(t *testing.T) {
	for _, tc := range []struct{ name, path string }{
		{"outside", "/outside/result.json"},
		{"sibling-prefix", "/safe-other/result.json"},
		{"relative", "result.json"},
		{"parent-traversal", "/safe/../outside/result.json"},
		{"uri", "file:///safe/result.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			records := artifactReadTestRecords(tc.path)
			event := artifactReadEventID(t, records)
			calls := 0
			c := newTranscriptArtifactReadCmdUsing(func(string, string) ([]ledgerRecord, string, string, error) { return records, "pi", "available", nil }, func(string, string) (artifactReadFile, error) { calls++; return nil, os.ErrPermission })
			c.SetArgs([]string{"session", "agent", "--event", event, "--snapshot", artifactReadTestSnapshot(t, records, event), "--finding", "1", "--allow-root", "/safe"})
			err := c.Execute()
			if err == nil || calls != 0 {
				t.Fatalf("ASSERT_ARTIFACT_REJECTED_PATH_BEFORE_OPENER_%s: calls=%d err=%v", tc.name, calls, err)
			}
		})
	}
}

func TestTranscriptArtifactReadWrongEventBeforeOpener(t *testing.T) {
	records := artifactReadTestRecords("/safe/result.json")
	calls := 0
	c := newTranscriptArtifactReadCmdUsing(func(string, string) ([]ledgerRecord, string, string, error) { return records, "pi", "available", nil }, func(string, string) (artifactReadFile, error) { calls++; return nil, os.ErrPermission })
	c.SetArgs([]string{"session", "agent", "--event", "wrong-event", "--snapshot", strings.Repeat("0", 64), "--finding", "1", "--allow-root", "/safe"})
	if err := c.Execute(); err == nil {
		t.Fatal("ASSERT_ARTIFACT_WRONG_EVENT_REJECTED")
	}
	if calls != 0 {
		t.Fatalf("ASSERT_ARTIFACT_WRONG_EVENT_BEFORE_OPENER: calls=%d", calls)
	}
}

func TestTranscriptArtifactReadAdmittedBoundedFile(t *testing.T) {
	requireTranscriptArtifactUnix(t)
	root := t.TempDir()
	var err error
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "result.json")
	want := bytes.Repeat([]byte("x"), transcriptArtifactReadMaxBytes+1)
	if err := os.WriteFile(path, want, 0600); err != nil {
		t.Fatal(err)
	}
	records := artifactReadTestRecords(path)
	event := artifactReadEventID(t, records)
	var out bytes.Buffer
	c := newTranscriptArtifactReadCmdUsing(func(string, string) ([]ledgerRecord, string, string, error) { return records, "pi", "available", nil }, openTranscriptArtifact)
	c.SetOut(&out)
	c.SetArgs([]string{"session", "agent", "--event", event, "--snapshot", artifactReadTestSnapshot(t, records, event), "--finding", "1", "--allow-root", root})
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	var got transcriptArtifactReadResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Coverage != "partial" || got.AcquiredBytes != transcriptArtifactReadMaxBytes {
		t.Fatalf("ASSERT_ARTIFACT_BOUNDED_PREFIX: coverage=%q bytes=%d", got.Coverage, got.AcquiredBytes)
	}
	decoded, err := base64.StdEncoding.DecodeString(got.ContentBase64)
	if err != nil || !bytes.Equal(decoded, want[:transcriptArtifactReadMaxBytes]) {
		t.Fatalf("ASSERT_ARTIFACT_BOUNDED_BASE64: err=%v", err)
	}
	sum := sha256.Sum256(decoded)
	if got.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("ASSERT_ARTIFACT_PREFIX_DIGEST: got=%s want=%x", got.SHA256, sum)
	}
}

type changingArtifactFile struct {
	io.Reader
	before, after os.FileInfo
	stats         int
}

func (f *changingArtifactFile) Stat() (os.FileInfo, error) {
	f.stats++
	if f.stats == 1 {
		return f.before, nil
	}
	return f.after, nil
}
func (*changingArtifactFile) Close() error { return nil }

func TestTranscriptArtifactReadReportsDetectableChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "changing")
	if err := os.WriteFile(path, []byte("a"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("longer"), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	root := "/safe"
	records := artifactReadTestRecords(root + "/result.json")
	event := artifactReadEventID(t, records)
	var out bytes.Buffer
	c := newTranscriptArtifactReadCmdUsing(func(string, string) ([]ledgerRecord, string, string, error) { return records, "pi", "available", nil }, func(string, string) (artifactReadFile, error) {
		return &changingArtifactFile{Reader: strings.NewReader("a"), before: before, after: after}, nil
	})
	c.SetOut(&out)
	c.SetArgs([]string{"session", "agent", "--event", event, "--snapshot", artifactReadTestSnapshot(t, records, event), "--finding", "1", "--allow-root", root})
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	var result transcriptArtifactReadResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil || result.Coverage != "unstable" {
		t.Fatalf("ASSERT_ARTIFACT_DETECTABLE_CHANGE: err=%v result=%+v", err, result)
	}
}

func TestTranscriptArtifactReadExactEventCLI(t *testing.T) {
	requireTranscriptArtifactUnix(t)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "result.json")
	if err := os.WriteFile(path, []byte("current-artifact"), 0600); err != nil {
		t.Fatal(err)
	}
	session, side := transcriptDiagnosticsFixture(t)
	source, err := os.ReadFile(side)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(source), "/unavailable/result.json", path, 1)
	if changed == string(source) {
		t.Fatal("ASSERT_ARTIFACT_CLI_FIXTURE")
	}
	if err := os.WriteFile(side, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	_, execute := setupNotebook(t)
	events, _ := ledgerAll(t, execute, session, "AAA", "--last", "1")
	if len(events) != 1 {
		t.Fatal("ASSERT_ARTIFACT_CLI_EVENT")
	}
	id, _ := events[0]["event_id"].(string)
	pageJSON, err := execute("transcript", "events", session, "AAA", "--diagnostics", "--event", id)
	if err != nil {
		t.Fatal(err)
	}
	var page struct {
		Snapshot string `json:"snapshot"`
	}
	if err := json.Unmarshal([]byte(pageJSON), &page); err != nil || page.Snapshot == "" {
		t.Fatalf("ASSERT_ARTIFACT_CLI_SNAPSHOT: %v", err)
	}
	out, err := execute("transcript", "artifact", "read", session, "AAA", "--event", id, "--snapshot", page.Snapshot, "--finding", "1", "--allow-root", root)
	if err != nil {
		t.Fatalf("ASSERT_ARTIFACT_CLI_ACQUISITION: %v", err)
	}
	var result transcriptArtifactReadResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	got, err := base64.StdEncoding.DecodeString(result.ContentBase64)
	if err != nil || string(got) != "current-artifact" || result.Coverage != "complete" || result.Snapshot != page.Snapshot {
		t.Fatalf("ASSERT_ARTIFACT_CLI_CURRENT_BYTES: err=%v result=%+v", err, result)
	}
}

func TestTranscriptArtifactReadExactBoundComplete(t *testing.T) {
	requireTranscriptArtifactUnix(t)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "exact")
	want := bytes.Repeat([]byte("x"), transcriptArtifactReadMaxBytes)
	if err := os.WriteFile(path, want, 0600); err != nil {
		t.Fatal(err)
	}
	records := artifactReadTestRecords(path)
	event := artifactReadEventID(t, records)
	var out bytes.Buffer
	c := newTranscriptArtifactReadCmdUsing(func(string, string) ([]ledgerRecord, string, string, error) { return records, "pi", "available", nil }, openTranscriptArtifact)
	c.SetOut(&out)
	c.SetArgs([]string{"session", "agent", "--event", event, "--snapshot", artifactReadTestSnapshot(t, records, event), "--finding", "1", "--allow-root", root})
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	var got transcriptArtifactReadResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(got.ContentBase64)
	if err != nil || got.Coverage != "complete" || got.AcquiredBytes != len(want) || !bytes.Equal(decoded, want) {
		t.Fatalf("ASSERT_ARTIFACT_EXACT_BOUND_COMPLETE: coverage=%s bytes=%d err=%v", got.Coverage, got.AcquiredBytes, err)
	}
}

func TestTranscriptArtifactReadFilesystemDenials(t *testing.T) {
	requireTranscriptArtifactUnix(t)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(outside, "secret")
	if err := os.WriteFile(secret, []byte("outside-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "linked-file")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked-root")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "directory"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, allowRoot, path, want string }{
		{"symlink-root", filepath.Join(root, "linked-root"), filepath.Join(root, "linked-root", "secret"), "denied"},
		{"symlink-leaf", root, filepath.Join(root, "linked-file"), "denied"},
		{"missing-leaf", root, filepath.Join(root, "absent"), "missing"},
		{"directory-leaf", root, filepath.Join(root, "directory"), "nonregular"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			records := artifactReadTestRecords(tc.path)
			event := artifactReadEventID(t, records)
			var out bytes.Buffer
			c := newTranscriptArtifactReadCmdUsing(func(string, string) ([]ledgerRecord, string, string, error) { return records, "pi", "available", nil }, openTranscriptArtifact)
			c.SetOut(&out)
			c.SilenceErrors = true
			c.SilenceUsage = true
			c.SetArgs([]string{"session", "agent", "--event", event, "--snapshot", artifactReadTestSnapshot(t, records, event), "--finding", "1", "--allow-root", tc.allowRoot})
			if err := c.Execute(); err == nil || !strings.Contains(err.Error(), tc.want) || out.Len() != 0 {
				t.Fatalf("ASSERT_ARTIFACT_FILESYSTEM_DENIAL_%s: err=%v output=%q", tc.name, err, out.String())
			}
		})
	}
}

func TestTranscriptArtifactReadRejectsSymlinkAndSwap(t *testing.T) {
	requireTranscriptArtifactUnix(t)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("outside-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	if _, err := openTranscriptArtifact(root, filepath.Join(root, "linked", "secret")); err == nil {
		t.Fatal("ASSERT_ARTIFACT_NOFOLLOW_INTERMEDIATE")
	}
	safe := filepath.Join(root, "slot")
	if err := os.Mkdir(safe, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(safe, "secret"), []byte("inside"), 0600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "slot", "secret")
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			_ = os.Rename(safe, filepath.Join(root, "moved"))
			_ = os.Symlink(outside, safe)
			time.Sleep(time.Microsecond)
			_ = os.Remove(safe)
			_ = os.Rename(filepath.Join(root, "moved"), safe)
		}
	}()
	for i := 0; i < 200; i++ {
		f, e := openTranscriptArtifact(root, target)
		if e != nil {
			continue
		}
		b, e := io.ReadAll(f)
		_ = f.Close()
		if e == nil && string(b) != "inside" {
			t.Fatalf("ASSERT_ARTIFACT_SWAP_NO_OUTSIDE_BYTES: %q", b)
		}
	}
	<-done
}

func requireTranscriptArtifactUnix(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("descriptor-relative artifact reads are unsupported on this platform")
	}
}

func artifactReadTestRecords(path string) []ledgerRecord {
	payload := `{"omitted":true,"reason":"bounded","contentBlocks":1,"rawResultBytes":9,"contentSummary":[{"textOmitted":true}],"fullResultPath":` + artifactReadJSONString(path) + `}`
	msg := `{"role":"toolResult","toolName":"test","content":[{"type":"text","text":` + artifactReadJSONString(payload) + `}]}`
	return []ledgerRecord{{Record: rawRecord{Type: "toolResult", UUID: "fixture", AgentID: "agent", Message: json.RawMessage(msg)}}}
}
func artifactReadJSONString(s string) string { b, _ := json.Marshal(s); return string(b) }
func artifactReadEventID(t *testing.T, records []ledgerRecord) string {
	t.Helper()
	events, err := projectLedger(records, "agent", []string{"identity", "message", "usage", "tools", "lifecycle"}, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events {
		if e["kind"] == "tool_result" {
			return stringValue(e["event_id"])
		}
	}
	t.Fatal("no tool result")
	return ""
}
func artifactReadTestSnapshot(t *testing.T, records []ledgerRecord, event string) string {
	t.Helper()
	events, err := projectLedger(records, "agent", []string{"identity", "message", "usage", "tools", "lifecycle"}, true)
	if err != nil {
		t.Fatal(err)
	}
	page, err := buildLedgerPageUsing("session", "agent", "pi", "available", []string{"identity", "message", "usage", "tools", "lifecycle"}, false, events, 1, "", event, false, nil, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	return page.Snapshot
}
