package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/valyala/fastjson"
)

// Experiment only: no production reader uses fastjson. This measures selected
// field extraction, not raw-payload preservation or complete observation.
type trialFields struct{ Type, Agent, Timestamp, Role string }

func trialStandard(b []byte) (trialFields, error) {
	var r rawRecord
	if err := json.Unmarshal(b, &r); err != nil {
		return trialFields{}, err
	}
	_, role := attentionRecordRole(ledgerRecord{Record: r})
	return trialFields{r.Type, r.AgentID, r.Timestamp, role}, nil
}
func trialFast(p *fastjson.Parser, b []byte) (trialFields, error) {
	v, err := p.ParseBytes(b)
	if err != nil {
		return trialFields{}, err
	}
	if v.Type() != fastjson.TypeObject && v.Type() != fastjson.TypeNull {
		return trialFields{}, fmt.Errorf("record is not object/null")
	}
	// Copy all strings before parser reuse. Get* alone is not a compatible
	// replacement for encoding/json's field matching and type validation.
	f := trialFields{Type: string(v.GetStringBytes("type")), Agent: string(v.GetStringBytes("agentId")), Timestamp: string(v.GetStringBytes("timestamp")), Role: "unknown"}
	m := v.Get("message")
	if m == nil || m.Type() != fastjson.TypeObject {
		return f, nil
	}
	f.Role = string(m.GetStringBytes("role"))
	if f.Role == "" {
		f.Role = f.Type
	}
	if f.Role == "" || f.Role == "message" {
		f.Role = "unknown"
	}
	if f.Role == "user" {
		for _, block := range m.GetArray("content") {
			if string(block.GetStringBytes("type")) == "tool_result" {
				f.Role = "toolResult"
				break
			}
		}
	}
	return f, nil
}
func TestFastjsonSemanticSurvey(t *testing.T) {
	var p fastjson.Parser
	for _, input := range []string{
		`{"type":"message","agentId":"A","message":{"role":"assistant"}}`,
		`{"agentId":"first","agentId":"last"}`, `{"AGENTID":"A"}`, `{"agentId":42}`,
		`null`, `[]`, `{"message":{"role":"user","content":[{"type":"tool_result"}]}}`,
	} {
		a, ae := trialStandard([]byte(input))
		b, be := trialFast(&p, []byte(input))
		t.Logf("input=%s equal=%t std=%+v fast=%+v std_error=%v fast_error=%v", input, a == b && (ae == nil) == (be == nil), a, b, ae, be)
	}
}
func TestFastjsonCopiedFields(t *testing.T) {
	var p fastjson.Parser
	first, err := trialFast(&p, []byte(`{"type":"message","agentId":"original","message":{"role":"assistant"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = trialFast(&p, []byte(`{"type":"another","agentId":"replaced","message":{"role":"toolResult"}}`)); err != nil {
		t.Fatal(err)
	}
	if first.Agent != "original" || first.Role != "assistant" || first.Type != "message" {
		t.Fatal("FASTJSON_COPY FAIL")
	}
	t.Log("FASTJSON_COPY PASS")
}
func trialCorpus(t testing.TB) [][]byte {
	t.Helper()
	path := os.Getenv("NN_FASTJSON_CORPUS")
	if path == "" {
		t.Skip("set NN_FASTJSON_CORPUS to frozen JSONL")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var lines [][]byte
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 65536), 16*1024*1024)
	for s.Scan() {
		lines = append(lines, bytes.Clone(s.Bytes()))
	}
	if err = s.Err(); err != nil {
		t.Fatal(err)
	}
	return lines
}
func TestFastjsonFrozenFields(t *testing.T) {
	lines := trialCorpus(t)
	var p fastjson.Parser
	for i, line := range lines {
		a, ae := trialStandard(line)
		b, be := trialFast(&p, line)
		if (ae == nil) != (be == nil) || (ae == nil && a != b) {
			t.Fatalf("FASTJSON_CORPUS FAIL: record %d std=%+v fast=%+v errors=%v/%v", i+1, a, b, ae, be)
		}
	}
	t.Logf("FASTJSON_CORPUS PASS: %d lines", len(lines))
}

var trialSink trialFields

func BenchmarkTranscriptFieldParser(b *testing.B) {
	lines := trialCorpus(b)
	size := int64(0)
	for _, line := range lines {
		size += int64(len(line))
	}
	for _, name := range []string{"encoding-json", "fastjson"} {
		b.Run(name, func(b *testing.B) {
			b.SetBytes(size)
			b.ReportAllocs()
			var p fastjson.Parser
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, line := range lines {
					if name == "fastjson" {
						trialSink, _ = trialFast(&p, line)
					} else {
						trialSink, _ = trialStandard(line)
					}
				}
			}
		})
	}
}
