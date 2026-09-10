package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func tailFixture(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "worker.jsonl")
	var b bytes.Buffer
	b.WriteString("{\"type\":\"session\"}\nmalformed\nnull\n")
	for i := 0; i < 400; i++ {
		role, body := "assistant", fmt.Sprintf(`[{"type":"toolCall","id":"c%d","name":"bash","arguments":{"command":"echo x"}}]`, i)
		if i%4 == 0 {
			role, body = "user", fmt.Sprintf(`"assignment %d"`, i)
		}
		fmt.Fprintf(&b, "{\"type\":\"message\",\"agentId\":\"A\",\"timestamp\":\"2026-09-10T00:00:00Z\",\"message\":{\"role\":%q,\"content\":%s}}\n", role, body)
	}
	if err := os.WriteFile(p, b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}
func assertTailNative(t *testing.T, p string, s observeTail) {
	t.Helper()
	source, err := captureTranscriptSource(p)
	if err != nil {
		t.Fatal(err)
	}
	var full []ledgerRecord
	for _, r := range ownedPiRecords(source.Records, "A", true) {
		full = append(full, ledgerRecord{Record: r, Path: p})
	}
	m, w, e, err := collectAttention(full, "available", "A", 100)
	if err != nil {
		t.Fatal(err)
	}
	am, aw, ae, err := collectAttention(s.records(), "available", "A", 100)
	if err != nil {
		t.Fatal(err)
	}
	if aw.Selected == aw.Requested {
		if earlier, ok := s.earlier(aw.FirstEvent, nil); ok {
			aw.Earlier = earlier
		}
	}
	if !reflect.DeepEqual(m, am) || !reflect.DeepEqual(w, aw) || !reflect.DeepEqual(e, ae) {
		t.Fatal("TAIL_NATIVE FAIL", w, aw)
	}
	context := attentionTaskContextFor("A", p, full, nil)
	bounded := attentionTaskContextFor("A", p, s.records(), nil)
	if context.Total != s.TaskTotal || context.Unavailable != s.TaskUnavailable || !reflect.DeepEqual(context.Evidence, bounded.Evidence) {
		t.Fatal("TAIL_CONTEXT FAIL")
	}
	if len(s.records()) > 103 {
		t.Fatal("TAIL_BOUND FAIL", len(s.records()))
	}
}
func TestObserveTailIntegration(t *testing.T) {
	p := writePiBackgroundSidechainFixture(t, t.TempDir())
	out, err := contextCommand(t, "observe", p)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Fields(strings.SplitN(out, "\n", 2)[0])[2]
	before, err := loadObserveState(id, p)
	if err != nil {
		t.Fatal(err)
	}
	var target observeLiveAgent
	for _, a := range before.Agents {
		if a.ID == "79d3f783-b96d-4c7" {
			target = a
		}
	}
	if target.TailSnapshot == "" || target.SourceStamp == nil {
		t.Fatal("TAIL_INTEGRATION FAIL: checkpoint absent")
	}
	f, err := os.OpenFile(target.SourceStamp.Path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("\n{\"type\":\"message\",\"agentId\":\"79d3f783-b96d-4c7\",\"message\":{\"role\":\"assistant\",\"content\":\"appended work\"}}\n")
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	out, err = contextCommand(t, "observe", p, "--refresh", id)
	if err != nil {
		t.Fatal(err)
	}
	id = strings.Fields(strings.SplitN(out, "\n", 2)[0])[2]
	after, err := loadObserveState(id, p)
	if err != nil {
		t.Fatal(err)
	}
	if target.SourceStamp.Stable && after.WorkerScans["verified_append"] < 1 {
		t.Fatal("TAIL_INTEGRATION FAIL: append path unused")
	}
	records, _, detail, err := ledgerRecords(p, "79d3f783-b96d-4c7")
	if err != nil {
		t.Fatal(err)
	}
	metrics, window, _, err := collectAttention(records, detail, "79d3f783-b96d-4c7", 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range after.Agents {
		if a.ID == "79d3f783-b96d-4c7" {
			if a.Room == nil || !reflect.DeepEqual(a.Room.Metrics, metrics) || !reflect.DeepEqual(a.Room.Window, window) {
				t.Fatalf("TAIL_INTEGRATION FAIL: native result differs: got=%+v expected metrics=%+v window=%+v", a.Room, metrics, window)
			}
			t.Log("TAIL_INTEGRATION PASS")
			return
		}
	}
	t.Fatal("TAIL_INTEGRATION FAIL: agent omitted")
}

func TestObserveTailCheckpoint(t *testing.T) {
	p := tailFixture(t)
	s, _, err := scanObserveTail(p, "A", 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	id, err := retainObserveTail(s)
	if err != nil || id == "" {
		t.Fatal(err)
	}
	loaded := loadObserveTail(id, p, "A", 100)
	if loaded == nil {
		t.Fatal("TAIL_CHECKPOINT FAIL")
	}
	if loadObserveTail(id, p, "B", 100) != nil || loadObserveTail(id, p, "A", 50) != nil || loadObserveTail(strings.Repeat("0", 64), p, "A", 100) != nil {
		t.Fatal("TAIL_CHECKPOINT FAIL: invalid binding accepted")
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("{\"type\":\"message\",\"agentId\":\"B\",\"message\":{\"role\":\"assistant\",\"content\":\"foreign work\"}}\n")
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	next, _, err := scanObserveTail(p, "A", 100, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if next.Owned != s.Owned || next.digest() != s.digest() {
		t.Fatal("TAIL_OWNERSHIP FAIL")
	}
	t.Log("TAIL_CHECKPOINT PASS; TAIL_OWNERSHIP PASS")
}

func TestObserveTailNative(t *testing.T) {
	p := tailFixture(t)
	s, incremental, err := scanObserveTail(p, "A", 100, nil)
	if err != nil || incremental {
		t.Fatal(err)
	}
	assertTailNative(t, p, s)
	t.Log("TAIL_NATIVE PASS; TAIL_CONTEXT PASS; TAIL_BOUND PASS")
}
func TestObserveTailAppend(t *testing.T) {
	p := tailFixture(t)
	prior, _, err := scanObserveTail(p, "A", 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !prior.Stamp.Stable {
		t.Skip("no stable identity metadata")
	}
	line := []byte("{\"type\":\"message\",\"agentId\":\"A\",\"message\":{\"role\":\"user\",\"content\":\"new assignment\"}}\n")
	appendBytes := func(b []byte) {
		f, e := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0600)
		if e != nil {
			t.Fatal(e)
		}
		_, e = f.Write(b)
		f.Close()
		if e != nil {
			t.Fatal(e)
		}
	}
	appendBytes(line[:len(line)-8])
	count := transcriptDecodeCount.Load()
	partial, inc, err := scanObserveTail(p, "A", 100, &prior)
	if err != nil || !inc || partial.Offset != prior.Offset || transcriptDecodeCount.Load() != count {
		t.Fatal("TAIL_PARTIAL FAIL", inc, err)
	}
	appendBytes(line[len(line)-8:])
	count = transcriptDecodeCount.Load()
	next, inc, err := scanObserveTail(p, "A", 100, &partial)
	if err != nil || !inc || transcriptDecodeCount.Load()-count != 1 {
		t.Fatal("TAIL_APPEND FAIL", inc, err)
	}
	assertTailNative(t, p, next)
	t.Log("TAIL_PARTIAL PASS; TAIL_APPEND PASS")
}
func TestObserveTailFallback(t *testing.T) {
	for _, kind := range []string{"rewrite", "truncate", "replace"} {
		t.Run(kind, func(t *testing.T) {
			p := tailFixture(t)
			prior, _, err := scanObserveTail(p, "A", 100, nil)
			if err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "rewrite":
				b = []byte(strings.ReplaceAll(string(b), "echo x", "echo y"))
			case "truncate":
				b = b[:len(b)/2]
			case "replace":
				if err = os.Rename(p, p+".old"); err != nil {
					t.Fatal(err)
				}
			}
			if err = os.WriteFile(p, b, 0600); err != nil {
				t.Fatal(err)
			}
			s, inc, err := scanObserveTail(p, "A", 100, &prior)
			if err != nil || inc {
				t.Fatal("TAIL_FALLBACK FAIL", err)
			}
			assertTailNative(t, p, s)
		})
	}
	t.Log("TAIL_FALLBACK PASS")
}
