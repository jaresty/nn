package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type captureFailWriter struct{}

func (captureFailWriter) Write(p []byte) (int, error) {
	return 0, fmt.Errorf("injected writer failure")
}

func TestCaptureStreamCompatibility(t *testing.T) {
	t.Log("procedure: TestCaptureStreamCompatibility; assertion: P5_CAPTURE_STREAM")
	for n := 0; n < 8; n++ {
		raw := bytes.Repeat([]byte{0, 128, '<', '&', '\n'}, n)
		c := &transcriptCapture{InputPath: "/tmp/<é>&", Path: "/tmp/quote\"root", Sources: map[string]capturedTranscriptSource{
			"z":    {Raw: raw, PrefixBytes: int64(len(raw)), Digest: captureHash(raw)},
			"a<&>": {Raw: []byte{}, Digest: captureHash(nil)},
			"nil":  {},
		}, AuthPaths: map[string]string{"k\x00a": "path"}}
		if n%2 == 1 {
			c.AgentSelection = []string{"A", "ROOT"}
		}
		want, e := json.Marshal(c)
		if e != nil {
			t.Fatal(e)
		}
		var got bytes.Buffer
		if e = writeCaptureJSON(&got, c); e != nil {
			t.Fatal(e)
		}
		if !bytes.Equal(want, got.Bytes()) {
			t.Fatalf("P5_CAPTURE_STREAM FAIL: encoding differs at padding case %d\nwant %s\ngot  %s", n, want, got.Bytes())
		}
		if e = writeCaptureJSON(captureFailWriter{}, c); e == nil {
			t.Fatal("P5_CAPTURE_STREAM FAIL: writer error suppressed")
		}
	}
	t.Log("P5_CAPTURE_STREAM PASS")
}

func TestAttentionDefaultCapture(t *testing.T) {
	t.Log("procedure: TestAttentionDefaultCapture; assertion: P10_DEFAULT_CAPTURE")
	t.Setenv("HOME", t.TempDir())
	path, id := attentionFixture(t, "pi")
	if _, e := buildAttention(path, []string{id}, "implementation"); e != nil {
		t.Fatal(e)
	}
	dir, e := captureCacheDir()
	if e != nil {
		t.Fatal(e)
	}
	files, e := filepath.Glob(filepath.Join(dir, "*.json"))
	if e != nil || len(files) != 1 {
		t.Fatalf("P10_DEFAULT_CAPTURE FAIL: missing normal capture %v", e)
	}
	b, e := os.ReadFile(files[0])
	if e != nil {
		t.Fatal(e)
	}
	var c transcriptCapture
	if e = json.Unmarshal(b, &c); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(c.AgentSelection, []string{id}) {
		t.Fatal("P10_DEFAULT_CAPTURE FAIL: selection requires separate enablement")
	}
	t.Log("P10_DEFAULT_CAPTURE PASS")
}

func TestAttentionScopedCaptureParity(t *testing.T) {
	t.Log("procedure: TestAttentionScopedCaptureParity; assertion: P2_CAPTURE_PARITY")
	dir := t.TempDir()
	root := filepath.Join(dir, "root.jsonl")
	content := `{"type":"session"}` + "\n"
	for _, id := range []string{"A", "B"} {
		side := filepath.Join(dir, "pi-subagents-test", "session", "tasks", id+".output")
		body := fmt.Sprintf(`{"type":"assistant","agentId":%q,"message":{"role":"assistant","content":[{"type":"toolCall","id":"cmd","name":"bash","arguments":{"command":"echo x"}}]}}`+"\n", id)
		if id == "B" {
			body += fmt.Sprintf(`{"type":"assistant","agentId":"B","message":{"role":"assistant","content":%q}}`+"\n", string(bytes.Repeat([]byte("x"), 100000)))
		}
		writeTranscriptFile(t, side, body)
		content += fmt.Sprintf(`{"type":"message","message":{"role":"toolResult","details":{"status":"background","agentId":%q,"fullOutputPath":%q}}}`+"\n", id, side)
	}
	writeTranscriptFile(t, root, content)
	full, e := newTranscriptCapture(root)
	if e != nil {
		t.Fatal(e)
	}
	scoped, e := newTranscriptCaptureForAgents(root, map[string]bool{"A": true})
	if e != nil {
		t.Fatal(e)
	}
	f, fs := full.ledger("A")
	s, ss := scoped.ledger("A")
	fm, fw, fe, e := collectAttention(f, fs, "A", 100)
	if e != nil {
		t.Fatal(e)
	}
	sm, sw, se, e := collectAttention(s, ss, "A", 100)
	if e != nil {
		t.Fatal(e)
	}
	if fm != sm || fw != sw || !reflect.DeepEqual(fe, se) {
		t.Fatal("P2_CAPTURE_PARITY FAIL: selected evidence differs")
	}
	// Unrelated source hydration cannot change the root-derived identity set.
	ft, e := buildPiTreeUsing(full.Path, full.read, full.resolve)
	if e != nil {
		t.Fatal(e)
	}
	st, e := buildPiTreeUsing(scoped.Path, scoped.read, scoped.resolve)
	if e != nil {
		t.Fatal(e)
	}
	ids := func(rows []agent) map[string]string {
		r := map[string]string{}
		for _, a := range rows {
			r[a.ID] = a.ParentID + ":" + a.Description
		}
		return r
	}
	if !reflect.DeepEqual(ids(ft), ids(st)) {
		t.Fatal("P2_CAPTURE_PARITY FAIL: identity population differs")
	}
	loaded, e := loadTranscriptCapture(scoped.ID)
	if e != nil {
		t.Fatal(e)
	}
	lr, ls := loaded.ledger("A")
	lm, lw, le, e := collectAttention(lr, ls, "A", 100)
	if e != nil || lm != sm || lw != sw || !reflect.DeepEqual(le, se) {
		t.Fatal("P2_CAPTURE_PARITY FAIL: loaded capture differs")
	}
	count := func(c *transcriptCapture) int64 {
		var n int64
		for _, s := range c.Sources {
			n += s.PrefixBytes
		}
		return n
	}
	t.Logf("capture measurement: full sources=%d bytes=%d; selected sources=%d bytes=%d", len(full.Sources), count(full), len(scoped.Sources), count(scoped))
	t.Log("P2_CAPTURE_PARITY PASS")
}
