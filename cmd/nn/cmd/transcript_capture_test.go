package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReviewCaptureSurvivesAppend(t *testing.T) {
	path := reviewFixture(t)
	first, e := buildReviewPage(path, "open-handoff", "canonical", "", 1, "")
	if e != nil {
		t.Fatal(e)
	}
	f, e := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = f.WriteString("{\"type\":\"custom\",\"customType\":\"subagents:record\",\"data\":{\"id\":\"C\",\"status\":\"completed\"}}\n")
	_ = f.Close()
	next, e := buildReviewPage(path, "open-handoff", "canonical", "", 1, first.NextCursor)
	if e != nil {
		t.Fatalf("retained cursor must survive append: %v", e)
	}
	if next.Rows[0].ID != "C" || next.Snapshot != first.Snapshot {
		t.Fatal("continuation must use original capture")
	}
	fresh, e := buildReviewPage(path, "open-handoff", "canonical", "", 1, "")
	if e != nil {
		t.Fatal(e)
	}
	if fresh.Eligible != 2 {
		t.Fatal("explicit refresh must observe appended return")
	}
}

func TestContextCaptureSurvivesDeletion(t *testing.T) {
	path := handoffFixture(t)
	first, e := buildTranscriptContext(path, "AAA", 2, 1, "")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Remove(path); e != nil {
		t.Fatal(e)
	}
	again, e := buildTranscriptContext(path, "AAA", 2, 1, first.Snapshot)
	if e != nil {
		t.Fatalf("retained context must survive source deletion: %v", e)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(again)
	if string(a) != string(b) {
		t.Fatal("cached context must be byte-identical")
	}
}

func TestCaptureWhileAppending(t *testing.T) {
	path := reviewFixture(t)
	f, e := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	stop, finished := make(chan struct{}), make(chan struct{})
	ready := make(chan struct{})
	go func() {
		defer close(finished)
		defer f.Close()
		first := true
		for {
			select {
			case <-stop:
				return
			default:
				_, _ = f.WriteString("{\"type\":\"message\",\"agentId\":\"B\",\"message\":{\"role\":\"assistant\",\"content\":\"append\"}}\n")
				if first {
					close(ready)
					first = false
				}
				time.Sleep(time.Millisecond)
			}
		}
	}()
	<-ready
	p, e := buildReviewTails(path, "open-handoff", "canonical", "", 1, "", 2, true, 1, "")
	close(stop)
	<-finished
	if e != nil {
		t.Fatalf("initial bundle must finish while source appends: %v", e)
	}
	again, e := buildReviewTails(path, "open-handoff", "canonical", "", 1, "", 2, true, 1, p.Snapshot)
	if e != nil {
		t.Fatal(e)
	}
	a, _ := json.Marshal(p)
	b, _ := json.Marshal(again)
	if !bytes.Equal(a, b) {
		t.Fatal("append must not alter retained bundle")
	}
}

func TestCapturePartialRecordAndCorruption(t *testing.T) {
	path := reviewFixture(t)
	f, e := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = f.WriteString("{\"type\":\"message\"")
	_ = f.Close()
	c, e := newTranscriptCapture(path)
	if e != nil {
		t.Fatal(e)
	}
	info, _ := os.Stat(path)
	if c.Sources[c.Path].PrefixBytes >= info.Size() {
		t.Fatal("unfinished record must be excluded")
	}
	dir, e := captureCacheDir()
	if e != nil {
		t.Fatal(e)
	}
	file := filepath.Join(dir, c.ID+".json")
	if e = os.WriteFile(file, []byte("corrupt"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = loadTranscriptCapture(c.ID); e == nil {
		t.Fatal("corrupt capture must fail closed")
	}
	if _, e = loadTranscriptCapture("../../bad"); e == nil {
		t.Fatal("invalid capture ID must reject")
	}
}

func TestCapturePreservesNativeWhitespace(t *testing.T) {
	path := reviewFixture(t)
	f, e := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = f.WriteString(`{"type":"message","agentId":"B","message": { "role": "assistant", "content": [ { "type": "text", "text": "native spacing" } ] }}
`)
	_ = f.Close()
	first, e := buildReviewTails(path, "open-handoff", "canonical", "", 1, "", 2, true, 1, "")
	if e != nil {
		t.Fatal(e)
	}
	again, e := buildReviewTails(path, "open-handoff", "canonical", "", 1, "", 2, true, 1, first.Snapshot)
	if e != nil {
		t.Fatalf("native whitespace must survive persisted replay: %v", e)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(again)
	if !bytes.Equal(a, b) {
		t.Fatal("native content-byte metadata must remain identical")
	}
}

func TestCapturedPageDoesNotReopenRawCapture(t *testing.T) {
	path := handoffFixture(t)
	first, e := buildTranscriptContext(path, "AAA", 2, 1, "")
	if e != nil {
		t.Fatal(e)
	}
	receipt := bundleEvents(t, first)[0]
	id := receipt["capture_id"].(string)
	dir, e := captureCacheDir()
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Remove(filepath.Join(dir, id+".json")); e != nil {
		t.Fatal(e)
	}
	again, e := buildTranscriptContext(path, "AAA", 2, 1, first.Snapshot)
	if e != nil {
		t.Fatalf("page replay must not reopen raw capture: %v", e)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(again)
	if !bytes.Equal(a, b) {
		t.Fatal("cached page changed")
	}
	raw, e := readCaptureFile(first.Snapshot + ".binding")
	if e != nil {
		t.Fatal(e)
	}
	var binding captureBinding
	if e = json.Unmarshal(raw, &binding); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(dir, binding.Pages[0]+".page"), []byte("corrupt"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = buildTranscriptContext(path, "AAA", 2, 1, first.Snapshot); e == nil {
		t.Fatal("corrupted encoded page must fail closed")
	}
}

func TestCaptureIndexMatchesOwnedLedger(t *testing.T) {
	path := reviewFixture(t)
	c, e := newTranscriptCapture(path)
	if e != nil {
		t.Fatal(e)
	}
	for _, id := range []string{"ROOT", "A", "B", "C", "D", "U"} {
		got, status := c.ledger(id)
		want, _, expected, e := ledgerRecords(path, id)
		if e != nil {
			t.Fatal(e)
		}
		a, _ := json.Marshal(got)
		b, _ := json.Marshal(want)
		if len(got) != len(want) || (len(got) > 0 && !bytes.Equal(a, b)) || status != expected {
			t.Fatalf("captured ownership differs for %s", id)
		}
	}
}
