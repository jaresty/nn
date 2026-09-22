package cmd

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestTranscriptArtifactDiagnosticsObservedEnvelope(t *testing.T) {
	// Sanitized shape of the retained successful LSP projection adapter response.
	text := `{"ok":true,"data":{"omitted":true,"reason":"Raw MCP result exceeded the details size limit and was replaced with this summary to keep session context bounded.","isError":false,"contentBlocks":1,"contentSummary":[{"type":"text","bytes":12970,"lines":1,"textOmitted":true}],"rawResultBytes":27290,"fullResultPath":"/fixture/mcp-output/mcp-result.txt","structuredContent":{"preservedFields":{"envelope_version":"1","tool":"lsp_trace_v2_structural_context","request_id":"fixture-projection","outcome":"COMPLETE","operation_status":"SUCCEEDED"}}}}`
	payload, err := json.Marshal(map[string]any{"content": []any{map[string]any{"type": "text", "text": text}}})
	if err != nil {
		t.Fatal(err)
	}
	r := diagnoseTranscriptArtifactEvent(ledgerEvent{"event_id": "observed-fixture", "kind": "tool_result", "payload": json.RawMessage(payload)})
	if r.EventID != "observed-fixture" || r.Status != transcriptArtifactDetected || !r.InspectionComplete || len(r.Issues) != 0 || len(r.Findings) != 1 {
		t.Fatalf("observed fixture: %#v", r)
	}
	f := r.Findings[0]
	if f.Kind != transcriptArtifactElision || f.OuterLocation != "/payload/content/0/text" || f.Location != "/data/omitted" || f.Artifact == nil || f.Artifact.RecordedPath != "/fixture/mcp-output/mcp-result.txt" || f.Artifact.Location != "/data/fullResultPath" || f.Artifact.Trust != "recorded_unverified" {
		t.Fatalf("observed provenance: %#v", f)
	}
}

func TestTranscriptArtifactDiagnosticsDetectsAdapterElision(t *testing.T) {
	e := ledgerEvent{"event_id": "e1", "kind": "tool_result", "payload": json.RawMessage(`"{\"omitted\":true,\"reason\":\"large\",\"contentBlocks\":1,\"contentSummary\":[{\"textOmitted\":true}],\"rawResultBytes\":10,\"fullResultPath\":\"/tmp/x\"}"`)}
	got := diagnoseTranscriptArtifactEvent(e)
	if got.Status != transcriptArtifactDetected || len(got.Findings) != 1 || got.Findings[0].Kind != transcriptArtifactElision {
		t.Fatalf("adapter elision: %#v", got)
	}
	if got.Findings[0].Artifact == nil || got.Findings[0].Artifact.RecordedPath != "/tmp/x" || got.Findings[0].Artifact.Trust != "recorded_unverified" {
		t.Fatalf("artifact reference: %#v", got.Findings[0].Artifact)
	}
}

func TestTranscriptArtifactDiagnosticsPreservesOuterInnerProvenance(t *testing.T) {
	inner := `{"omitted":true,"reason":"x","contentBlocks":1,"contentSummary":[{"textOmitted":true}],"rawResultBytes":1}`
	e := ledgerEvent{"event_id": "e2", "kind": "tool_result", "payload": json.RawMessage(`{"content":"` + escapeJSON(inner) + `"}`)}
	got := diagnoseTranscriptArtifactEvent(e)
	if len(got.Findings) != 1 || got.Findings[0].OuterLocation != "/payload/content" || got.Findings[0].Location != "/omitted" {
		t.Fatalf("provenance: %#v", got)
	}
	if got.Findings[0].Artifact != nil {
		t.Fatalf("unexpected artifact: %#v", got.Findings[0].Artifact)
	}
}

func TestTranscriptArtifactDiagnosticsWrappedProvenance(t *testing.T) {
	inner := `{"ok":true,"data":{"omitted":true,"reason":"x","contentBlocks":1,"contentSummary":[{"textOmitted":true}],"rawResultBytes":1,"fullResultPath":"p"}}`
	e := ledgerEvent{"event_id": "wrap", "kind": "tool_result", "payload": json.RawMessage(`{"content":[{"type":"text","text":` + artifactJSONString(inner) + `}]}`)}
	got := diagnoseTranscriptArtifactEvent(e)
	if len(got.Findings) != 1 || got.Findings[0].OuterLocation != "/payload/content/0/text" || got.Findings[0].Location != "/data/omitted" {
		t.Fatalf("wrapped provenance: %#v", got)
	}
	if got.Findings[0].Artifact == nil || got.Findings[0].Artifact.Location != "/data/fullResultPath" {
		t.Fatalf("wrapped reference: %#v", got.Findings[0].Artifact)
	}
}

func TestTranscriptArtifactDiagnosticsDetectsMCPMarkerOnly(t *testing.T) {
	e := ledgerEvent{"event_id": "e3", "kind": "tool_result", "payload": "prefix [MCP text output truncated: see artifact]"}
	got := diagnoseTranscriptArtifactEvent(e)
	if len(got.Findings) != 1 || got.Findings[0].Kind != transcriptArtifactTruncation || got.Findings[0].TextOffset <= 0 {
		t.Fatalf("marker: %#v", got)
	}
}

func TestTranscriptArtifactDiagnosticsRejectsLookalikesAndWrongEvents(t *testing.T) {
	cases := []ledgerEvent{
		{"event_id": "a", "kind": "assistant", "payload": `{"omitted":true}`},
		{"event_id": "b", "kind": "tool_result", "payload": `{"fullOutputPath":"x"}`},
		{"event_id": "c", "kind": "tool_result", "payload": "[truncated 4 chars]"},
	}
	for i, e := range cases {
		got := diagnoseTranscriptArtifactEvent(e)
		want := transcriptArtifactNotDetected
		if i == 2 {
			want = transcriptArtifactUninspected
		}
		if got.Status != want || len(got.Findings) != 0 {
			t.Fatalf("false positive for %#v: %#v", e, got)
		}
	}
}

func TestTranscriptArtifactDiagnosticsMalformedAndBounds(t *testing.T) {
	tooLong := make([]byte, 65537)
	for i := range tooLong {
		tooLong[i] = 'x'
	}
	got := diagnoseTranscriptArtifactEvent(ledgerEvent{"event_id": "x", "kind": "tool_result", "payload": string(tooLong)})
	if got.Status != transcriptArtifactUninspected || got.InspectionComplete {
		t.Fatalf("bound: %#v", got)
	}
}

func TestTranscriptArtifactDiagnosticsDoesNotMutateInput(t *testing.T) {
	raw := json.RawMessage(`{"content":"[MCP text output truncated: x]"}`)
	e := ledgerEvent{"event_id": "x", "kind": "tool_result", "payload": raw}
	before := append([]byte(nil), raw...)
	diagnoseTranscriptArtifactEvent(e)
	if !reflect.DeepEqual(raw, json.RawMessage(before)) {
		t.Fatal("input mutated")
	}
}

func TestTranscriptArtifactDiagnosticsJSONContractAndPaths(t *testing.T) {
	base := func(path string) string {
		suffix := ""
		if path != "" {
			suffix = `,"fullResultPath":` + path
		}
		return `{"omitted":true,"reason":"x","contentBlocks":1,"contentSummary":[{"textOmitted":true}],"rawResultBytes":1` + suffix + `}`
	}
	for name, path := range map[string]string{"missing": "", "null": "null", "empty": "\"\"", "nonstring": "7"} {
		t.Run(name, func(t *testing.T) {
			text := base(path)
			e := ledgerEvent{"event_id": name, "kind": "tool_result", "payload": json.RawMessage(`{"content":[{"type":"text","text":` + artifactJSONString(text) + `}]}`)}
			r := diagnoseTranscriptArtifactEvent(e)
			if r.Status != transcriptArtifactDetected || !r.InspectionComplete || len(r.Issues) != 0 || len(r.Findings) != 1 {
				t.Fatalf("result: %#v", r)
			}
			f := r.Findings[0]
			if f.OuterLocation != "/payload/content/0/text" || f.Location != "/omitted" || f.Artifact != nil {
				t.Fatalf("path: %#v", f)
			}
		})
	}
	for name, wrapped := range map[string]string{"direct": `{"omitted":true,"reason":"x","contentBlocks":1,"contentSummary":[{"textOmitted":true}],"rawResultBytes":1,"fullResultPath":"../untrusted path"}`, "wrapped": `{"ok":false,"data":{"omitted":true,"reason":"x","contentBlocks":1,"contentSummary":[{"textOmitted":true}],"rawResultBytes":1,"fullResultPath":"../untrusted path"}}`} {
		t.Run(name, func(t *testing.T) {
			e := ledgerEvent{"event_id": name, "kind": "tool_result", "payload": json.RawMessage(`{"content":[{"type":"text","text":` + artifactJSONString(wrapped) + `}]}`)}
			r := diagnoseTranscriptArtifactEvent(e)
			if r.Status != transcriptArtifactDetected || !r.InspectionComplete || len(r.Issues) != 0 || len(r.Findings) != 1 {
				t.Fatalf("result: %#v", r)
			}
			f := r.Findings[0]
			b, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var m map[string]any
			if err = json.Unmarshal(b, &m); err != nil {
				t.Fatal(err)
			}
			want := "/fullResultPath"
			if name == "wrapped" {
				want = "/data/fullResultPath"
			}
			if f.OuterLocation != "/payload/content/0/text" || f.Location != strings.TrimSuffix(want, "/fullResultPath")+"/omitted" || f.Artifact == nil || f.Artifact.RecordedPath != "../untrusted path" || f.Artifact.Location != want || f.Artifact.Trust != "recorded_unverified" {
				t.Fatalf("%#v", f)
			}
			fm := m["findings"].([]any)[0].(map[string]any)
			am := fm["artifact"].(map[string]any)
			for k, v := range map[string]string{"recorded_path": "../untrusted path", "location": want, "trust": "recorded_unverified"} {
				if am[k] != v {
					t.Fatalf("wire %s=%v", k, am[k])
				}
			}
			for _, k := range []string{"RecordedPath", "Location", "Trust"} {
				if _, ok := am[k]; ok {
					t.Fatalf("capitalized %s", k)
				}
			}
			if fm["outer_location"] != "/payload/content/0/text" || fm["location"] != f.Location {
				t.Fatalf("finding wire %#v", fm)
			}
		})
	}
	for name, e := range map[string]ledgerEvent{"detected": {"event_id": "d", "kind": "tool_result", "payload": "[MCP text output truncated: x]"}, "not": {"event_id": "n", "kind": "assistant", "payload": "x"}, "uninspected": {"event_id": "u", "kind": "tool_result", "payload": json.RawMessage(strings.Repeat("x", 4*1024*1024+1))}} {
		t.Run("wire-"+name, func(t *testing.T) {
			b, err := json.Marshal(diagnoseTranscriptArtifactEvent(e))
			if err != nil {
				t.Fatal(err)
			}
			var m map[string]any
			if err = json.Unmarshal(b, &m); err != nil {
				t.Fatal(err)
			}
			wantStatus := map[string]string{"detected": transcriptArtifactDetected, "not": transcriptArtifactNotDetected, "uninspected": transcriptArtifactUninspected}[name]
			if m["version"] != transcriptArtifactVersion || m["event_id"] != e["event_id"] || m["status"] != wantStatus {
				t.Fatalf("header %#v", m)
			}
			for _, k := range []string{"findings", "issues"} {
				if _, ok := m[k].([]any); !ok {
					t.Fatalf("%s %#v", k, m[k])
				}
			}
			if name == "detected" {
				f := m["findings"].([]any)[0].(map[string]any)
				if f["kind"] != "upstream_truncation" || f["text"] != "[MCP text output truncated: x]" || f["location"] != "/payload" || f["outer_location"] != "/payload" || f["text_offset"] != float64(0) {
					t.Fatalf("finding %#v", f)
				}
				if _, ok := f["artifact"]; ok {
					t.Fatal("artifact")
				}
			}
			if name == "uninspected" {
				is := m["issues"].([]any)[0].(map[string]any)
				if is["location"] != "/payload" || is["reason"] != "raw payload exceeds 4 MiB admission bound" {
					t.Fatalf("issue %#v", is)
				}
				for _, k := range []string{"Location", "Reason"} {
					if _, ok := is[k]; ok {
						t.Fatal(k)
					}
				}
			}
		})
	}
}

func TestTranscriptArtifactDiagnosticsBoundaries(t *testing.T) {
	for _, n := range []int{65536, 65537} {
		s := strings.Repeat("x", n)
		r := diagnoseTranscriptArtifactEvent(ledgerEvent{"event_id": "x", "kind": "tool_result", "payload": map[string]any{"content": s}})
		if n == 65536 && r.Status != transcriptArtifactNotDetected {
			t.Fatalf("candidate equality: %#v", r)
		}
		if n == 65537 && r.Status != transcriptArtifactUninspected {
			t.Fatalf("candidate overflow: %#v", r)
		}
	}
	for _, n := range []int{32, 33} {
		bs := make([]any, n)
		for i := range bs {
			bs[i] = map[string]any{"type": "text", "text": "x"}
		}
		r := diagnoseTranscriptArtifactEvent(ledgerEvent{"event_id": "x", "kind": "tool_result", "payload": map[string]any{"content": bs}})
		if n == 32 && (!r.InspectionComplete || len(r.Issues) != 0) {
			t.Fatalf("block equality: %#v", r)
		}
		if n == 33 && (r.InspectionComplete || !hasIssue(r, "content block limit")) {
			t.Fatalf("block overflow: %#v", r)
		}
	}
	for _, n := range []int{65536, 65537} {
		prefix := `{"x":"`
		s := prefix + strings.Repeat("a", n-len(prefix)-2) + `"}`
		r := diagnoseTranscriptArtifactEvent(ledgerEvent{"event_id": "x", "kind": "tool_result", "payload": map[string]any{"content": s}})
		if n == 65536 && (!r.InspectionComplete || len(r.Issues) != 0) {
			t.Fatalf("candidate equality: %#v", r)
		}
		if n == 65537 && (r.InspectionComplete || !hasIssue(r, "64 KiB")) {
			t.Fatalf("candidate overflow: %#v", r)
		}
	}
}

func TestTranscriptArtifactDiagnosticsFindingsLimitAndOrdering(t *testing.T) {
	blocks := make([]any, 17)
	for i := range blocks {
		blocks[i] = map[string]any{"type": "text", "text": `{"omitted":true,"reason":"x","contentBlocks":1,"contentSummary":[{"textOmitted":true}],"rawResultBytes":1}`}
	}
	for _, n := range []int{16, 17} {
		r := diagnoseTranscriptArtifactEvent(ledgerEvent{"event_id": "f", "kind": "tool_result", "payload": map[string]any{"content": blocks[:n]}})
		want := n
		if want > 16 {
			want = 16
		}
		if len(r.Findings) != want {
			t.Fatalf("n=%d: %#v", n, r)
		}
		if n == 16 && (!r.InspectionComplete || len(r.Issues) != 0) {
			t.Fatalf("equality: %#v", r)
		}
		if n == 17 && (r.InspectionComplete || !hasIssue(r, "finding limit")) {
			t.Fatalf("overflow: %#v", r)
		}
	}
}
func TestTranscriptArtifactDiagnosticsWrapperDiscriminators(t *testing.T) {
	base := `{"omitted":true,"reason":"x","contentBlocks":1,"contentSummary":[{"textOmitted":true}],"rawResultBytes":1}`
	for name, s := range map[string]string{"true": `{"ok":true,"data":` + base + `}`, "false": `{"ok":false,"data":` + base + `}`, "missing": `{"data":` + base + `}`, "wrong": `{"ok":"yes","data":` + base + `}`} {
		t.Run(name, func(t *testing.T) {
			r := diagnoseTranscriptArtifactEvent(ledgerEvent{"event_id": name, "kind": "tool_result", "payload": map[string]any{"content": s}})
			if (len(r.Findings) > 0) != (name == "true" || name == "false") {
				t.Fatalf("%s: %#v", name, r)
			}
		})
	}
}
func TestTranscriptArtifactDiagnosticsUnsupportedAndMapImmutability(t *testing.T) {
	m := map[string]any{"content": []any{map[string]any{"type": "text", "text": "x"}}}
	before := fmt.Sprintf("%#v", m)
	r := diagnoseTranscriptArtifactEvent(ledgerEvent{"event_id": "m", "kind": "tool_result", "payload": m})
	if r.Status != transcriptArtifactNotDetected || fmt.Sprintf("%#v", m) != before {
		t.Fatalf("map: %#v", r)
	}
	r = diagnoseTranscriptArtifactEvent(ledgerEvent{"event_id": "u", "kind": "tool_result", "payload": 42})
	if r.Status != transcriptArtifactUninspected || r.InspectionComplete {
		t.Fatalf("unsupported: %#v", r)
	}
}
func TestTranscriptArtifactDiagnosticsRawSizeA(t *testing.T) {
	for _, n := range []int{4 * 1024 * 1024, 4*1024*1024 + 1} {
		pad := n - len(`{"content":[{"type":"text","text":"x"}],"p":""}`)
		raw := json.RawMessage(`{"content":[{"type":"text","text":"x"}],"p":"` + strings.Repeat(" ", pad) + `"}`)
		if len(raw) != n {
			t.Fatalf("size %d", len(raw))
		}
		r := diagnoseTranscriptArtifactEvent(ledgerEvent{"event_id": "r", "kind": "tool_result", "payload": raw})
		if n == 4*1024*1024 && (!r.InspectionComplete || len(r.Issues) != 0) {
			t.Fatalf("equal %#v", r)
		}
		if n > 4*1024*1024 && (r.InspectionComplete || !hasIssue(r, "4 MiB")) {
			t.Fatalf("over %#v", r)
		}
	}
}
func TestTranscriptArtifactDiagnosticsDepthB(t *testing.T) {
	for _, n := range []int{16, 17} {
		p := `{"content":[{"type":"text","text":"x"}],"u":`
		s := p + strings.Repeat(`{"u":`, n-1) + `0` + strings.Repeat(`}`, n-1) + `}`
		r := diagnoseTranscriptArtifactEvent(ledgerEvent{"event_id": "d", "kind": "tool_result", "payload": map[string]any{"content": s}})
		if n == 16 && !r.InspectionComplete {
			t.Fatalf("d16 %#v", r)
		}
		if n == 17 && (r.InspectionComplete || !hasIssue(r, "depth")) {
			t.Fatalf("d17 %#v", r)
		}
	}
}
func TestTranscriptArtifactDiagnosticsDeepMapImmutability(t *testing.T) {
	m := map[string]any{"content": []any{map[string]any{"type": "text", "text": `{"omitted":true,"reason":"x","contentBlocks":1,"contentSummary":[{"textOmitted":true}],"rawResultBytes":1}`}}, "nested": map[string]any{"x": []any{1, "q"}}}
	b, _ := json.Marshal(m)
	before := append([]byte(nil), b...)
	diagnoseTranscriptArtifactEvent(ledgerEvent{"event_id": "m", "kind": "tool_result", "payload": m})
	after, _ := json.Marshal(m)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("map mutated")
	}
}

func hasIssue(r transcriptArtifactDiagnostic, fragment string) bool {
	for _, i := range r.Issues {
		if strings.Contains(i.Reason, fragment) {
			return true
		}
	}
	return false
}
func escapeJSON(s string) string         { b, _ := json.Marshal(s); return string(b[1 : len(b)-1]) }
func artifactJSONString(s string) string { b, _ := json.Marshal(s); return string(b) }

func TestTranscriptArtifactDiagnosticsDepthProposalCandidateAndRawBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		event     ledgerEvent
		status    string
		complete  bool
		issueText string
	}{
		{
			name:     "candidate-depth-16-accepted",
			event:    artifactDepthProposalCandidateEvent(16, false),
			status:   transcriptArtifactNotDetected,
			complete: true,
		},
		{
			name:      "candidate-depth-17-rejected",
			event:     artifactDepthProposalCandidateEvent(17, false),
			status:    transcriptArtifactUninspected,
			complete:  false,
			issueText: "depth",
		},
		{
			name:     "raw-depth-16-accepted",
			event:    artifactDepthProposalRawEvent(16, false),
			status:   transcriptArtifactNotDetected,
			complete: true,
		},
		{
			name:      "raw-depth-17-rejected",
			event:     artifactDepthProposalRawEvent(17, false),
			status:    transcriptArtifactUninspected,
			complete:  false,
			issueText: "depth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diagnoseTranscriptArtifactEvent(tt.event)
			if got.Status != tt.status {
				t.Fatalf("status = %q, want %q; result = %#v", got.Status, tt.status, got)
			}
			if len(got.Findings) != 0 {
				t.Fatalf("findings = %#v, want none", got.Findings)
			}
			if got.InspectionComplete != tt.complete {
				t.Fatalf("inspection_complete = %v, want %v; result = %#v", got.InspectionComplete, tt.complete, got)
			}
			if tt.issueText != "" && !artifactDepthProposalHasIssue(got, tt.issueText) {
				t.Fatalf("issues = %#v, want an issue containing %q", got.Issues, tt.issueText)
			}
			if tt.issueText == "" && len(got.Issues) != 0 {
				t.Fatalf("issues = %#v, want none", got.Issues)
			}
		})
	}
}

func TestTranscriptArtifactDiagnosticsDepthProposalEscapedDelimiters(t *testing.T) {
	for _, tt := range []struct {
		name  string
		event ledgerEvent
	}{
		{"candidate", artifactDepthProposalCandidateEvent(16, true)},
		{"raw", artifactDepthProposalRawEvent(16, true)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := diagnoseTranscriptArtifactEvent(tt.event)
			if got.Status != transcriptArtifactNotDetected || !got.InspectionComplete || len(got.Findings) != 0 || len(got.Issues) != 0 {
				t.Fatalf("escaped delimiters changed depth result: %#v", got)
			}
		})
	}
}

// artifactDepthProposalCandidateEvent keeps the outer map shallow so the
// recognizer's candidate-text bound, rather than the raw-payload bound, is
// what establishes the candidate fixture's depth.
func artifactDepthProposalCandidateEvent(depth int, escaped bool) ledgerEvent {
	candidate := artifactDepthProposalNestedObject(depth, escaped)
	return ledgerEvent{
		"event_id": "depth-proposal-candidate",
		"kind":     "tool_result",
		"payload":  map[string]any{"content": candidate},
	}
}

// artifactDepthProposalRawEvent puts only a small supported text block at the
// root and places the independently constructed depth under an unused field.
func artifactDepthProposalRawEvent(depth int, escaped bool) ledgerEvent {
	payload := artifactDepthProposalRawPayload(depth, escaped)
	return ledgerEvent{
		"event_id": "depth-proposal-raw",
		"kind":     "tool_result",
		"payload":  json.RawMessage(payload),
	}
}

func artifactDepthProposalNestedObject(depth int, escaped bool) string {
	if depth < 1 {
		panic(fmt.Sprintf("invalid proposal depth %d", depth))
	}
	prefix := strings.Repeat(`{"unused":`, depth-1)
	value := "0"
	if escaped {
		value = fmt.Sprintf("%q", `ignored { [ \"quoted\" ] }`)
	}
	return `{"content":"plain","unused":` + prefix + value + strings.Repeat("}", depth-1) + `}`
}

func artifactDepthProposalRawPayload(depth int, escaped bool) []byte {
	if depth < 2 {
		panic(fmt.Sprintf("raw proposal depth must be at least 2, got %d", depth))
	}
	unused := artifactDepthProposalNestedObject(depth-1, escaped)
	return []byte(`{"content":[{"type":"text","text":"plain"}],"unused":` + unused + `}`)
}

func artifactDepthProposalHasIssue(r transcriptArtifactDiagnostic, fragment string) bool {
	for _, issue := range r.Issues {
		if strings.Contains(issue.Reason, fragment) {
			return true
		}
	}
	return false
}

func artifactNegativeProposalEnvelope() map[string]any {
	return map[string]any{"omitted": true, "reason": "bounded", "contentBlocks": 1, "contentSummary": []any{map[string]any{"textOmitted": true}}, "rawResultBytes": 1}
}

func artifactNegativeProposalText(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func artifactNegativeProposalEvent(kind string, texts ...string) ledgerEvent {
	blocks := make([]any, len(texts))
	for i, text := range texts {
		blocks[i] = map[string]any{"type": "text", "text": text}
	}
	return ledgerEvent{"event_id": "proposal", "kind": kind, "payload": map[string]any{"content": blocks}}
}

func TestTranscriptArtifactDiagnosticsNegativeProposalDiscriminators(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"missing-omitted", func(v map[string]any) { delete(v, "omitted") }},
		{"wrong-omitted-type", func(v map[string]any) { v["omitted"] = "true" }},
		{"omitted-false", func(v map[string]any) { v["omitted"] = false }},
		{"missing-reason", func(v map[string]any) { delete(v, "reason") }},
		{"wrong-reason-type", func(v map[string]any) { v["reason"] = 7 }},
		{"empty-reason", func(v map[string]any) { v["reason"] = "" }},
		{"missing-contentBlocks", func(v map[string]any) { delete(v, "contentBlocks") }},
		{"wrong-contentBlocks-type", func(v map[string]any) { v["contentBlocks"] = "1" }},
		{"negative-contentBlocks", func(v map[string]any) { v["contentBlocks"] = -1 }},
		{"fractional-contentBlocks", func(v map[string]any) { v["contentBlocks"] = 1.5 }},
		{"missing-contentSummary", func(v map[string]any) { delete(v, "contentSummary") }},
		{"wrong-contentSummary-type", func(v map[string]any) { v["contentSummary"] = "omitted" }},
		{"missing-textOmitted", func(v map[string]any) { v["contentSummary"] = []any{map[string]any{"type": "text"}} }},
		{"textOmitted-false", func(v map[string]any) { v["contentSummary"] = []any{map[string]any{"textOmitted": false}} }},
		{"missing-rawResultBytes", func(v map[string]any) { delete(v, "rawResultBytes") }},
		{"wrong-rawResultBytes-type", func(v map[string]any) { v["rawResultBytes"] = "1" }},
		{"negative-rawResultBytes", func(v map[string]any) { v["rawResultBytes"] = -1 }},
		{"fractional-rawResultBytes", func(v map[string]any) { v["rawResultBytes"] = 1.5 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := artifactNegativeProposalEnvelope()
			tc.mutate(v)
			r := diagnoseTranscriptArtifactEvent(artifactNegativeProposalEvent("tool_result", artifactNegativeProposalText(v)))
			if r.Status != transcriptArtifactNotDetected || !r.InspectionComplete || len(r.Findings) != 0 || len(r.Issues) != 0 {
				t.Fatalf("%s: %#v", tc.name, r)
			}
		})
	}
}

func TestTranscriptArtifactDiagnosticsNegativeProposalPathsAndNonRecursiveWrappers(t *testing.T) {
	base := artifactNegativeProposalEnvelope()
	cases := []struct {
		name string
		v    any
	}{
		{"lone-fullOutputPath", map[string]any{"fullOutputPath": "/tmp/pi.output"}},
		{"lone-fullResultPath", map[string]any{"fullResultPath": "/tmp/result"}},
		{"nested-valid-envelope", map[string]any{"nested": base}},
		{"nested-ok-data-wrappers", map[string]any{"ok": true, "data": map[string]any{"ok": true, "data": base}}},
		{"nested-json-string", map[string]any{"nested": artifactNegativeProposalText(base)}},
		{"two-valid-wrappers", map[string]any{"first": map[string]any{"ok": true, "data": base}, "second": map[string]any{"ok": true, "data": base}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := diagnoseTranscriptArtifactEvent(artifactNegativeProposalEvent("tool_result", artifactNegativeProposalText(tc.v)))
			if r.Status != transcriptArtifactNotDetected || !r.InspectionComplete || len(r.Findings) != 0 || len(r.Issues) != 0 {
				t.Fatalf("%s: %#v", tc.name, r)
			}
		})
	}
}

func TestTranscriptArtifactDiagnosticsNegativeProposalNonToolResult(t *testing.T) {
	valid := artifactNegativeProposalText(artifactNegativeProposalEnvelope())
	for _, kind := range []string{"user", "assistant"} {
		r := diagnoseTranscriptArtifactEvent(artifactNegativeProposalEvent(kind, valid, "[MCP text output truncated: quoted marker]"))
		if r.Status != transcriptArtifactNotDetected || !r.InspectionComplete || len(r.Findings) != 0 || len(r.Issues) != 0 {
			t.Fatalf("kind=%s: %#v", kind, r)
		}
	}
}

func TestTranscriptArtifactDiagnosticsOrderingProposal(t *testing.T) {
	base := artifactNegativeProposalText(artifactNegativeProposalEnvelope())
	for _, count := range []int{16, 17} {
		texts := make([]string, count)
		for i := range texts {
			texts[i] = base
		}
		r := diagnoseTranscriptArtifactEvent(artifactNegativeProposalEvent("tool_result", texts...))
		want := count
		if want > 16 {
			want = 16
		}
		if r.Status != transcriptArtifactDetected || len(r.Findings) != want {
			t.Fatalf("count=%d: %#v", count, r)
		}
		for i, f := range r.Findings {
			if f.Kind != transcriptArtifactElision || f.OuterLocation != fmt.Sprintf("/payload/content/%d/text", i) || f.Location != "/omitted" {
				t.Fatalf("count=%d index=%d: %#v", count, i, f)
			}
		}
		if count == 16 {
			if !r.InspectionComplete || len(r.Issues) != 0 {
				t.Fatalf("equality: %#v", r)
			}
		} else if r.InspectionComplete || !artifactNegativeProposalHasIssue(r, "finding limit exceeded") {
			t.Fatalf("overflow: %#v", r)
		}
	}
}

func artifactNegativeProposalHasIssue(r transcriptArtifactDiagnostic, reason string) bool {
	for _, issue := range r.Issues {
		if issue.Reason == reason {
			return true
		}
	}
	return false
}

func TestTranscriptArtifactDiagnosticsBlockCandidateBoundary(t *testing.T) {
	for _, n := range []int{65536, 65537} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			prefix, suffix := `{"pad":"`, `"}`
			candidate := prefix + strings.Repeat("x", n-len(prefix)-len(suffix)) + suffix
			if len(candidate) != n {
				t.Fatalf("candidate length = %d, want %d", len(candidate), n)
			}
			raw, err := json.Marshal(map[string]any{"content": []any{map[string]any{"type": "text", "text": candidate}}})
			if err != nil {
				t.Fatal(err)
			}
			if len(raw) >= 4*1024*1024 {
				t.Fatalf("raw payload length = %d, want < 4 MiB", len(raw))
			}
			var shape map[string]json.RawMessage
			if err := json.Unmarshal(raw, &shape); err != nil {
				t.Fatal(err)
			}
			var blocks []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if err := json.Unmarshal(shape["content"], &blocks); err != nil {
				t.Fatal(err)
			}
			if len(blocks) != 1 || blocks[0].Type != "text" || len(blocks[0].Text) != n {
				t.Fatalf("serialized block text shape/length invalid: %#v", blocks)
			}

			got := diagnoseTranscriptArtifactEvent(ledgerEvent{
				"event_id": "block-candidate-proposal",
				"kind":     "tool_result",
				"payload":  json.RawMessage(raw),
			})
			if n == 65536 {
				if got.Status != transcriptArtifactNotDetected || !got.InspectionComplete || len(got.Findings) != 0 || len(got.Issues) != 0 {
					t.Fatalf("block candidate equality: %#v", got)
				}
				return
			}
			if got.Status != transcriptArtifactUninspected || got.InspectionComplete || len(got.Findings) != 0 || len(got.Issues) != 1 || got.Issues[0].Location != "/payload/content/0/text" || !strings.Contains(got.Issues[0].Reason, "64 KiB") {
				t.Fatalf("block candidate overflow: %#v", got)
			}
		})
	}
}
