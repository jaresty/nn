package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	transcriptArtifactVersion     = "nn.transcript.artifact-diagnostics/v1"
	transcriptArtifactElision     = "upstream_elision"
	transcriptArtifactTruncation  = "upstream_truncation"
	transcriptArtifactDetected    = "detected"
	transcriptArtifactNotDetected = "not_detected"
	transcriptArtifactUninspected = "uninspected"
)

type transcriptArtifactDiagnostic struct {
	Version            string                      `json:"version"`
	EventID            string                      `json:"event_id"`
	Status             string                      `json:"status"`
	Findings           []transcriptArtifactFinding `json:"findings"`
	InspectionComplete bool                        `json:"inspection_complete"`
	Issues             []transcriptArtifactIssue   `json:"issues"`
}
type transcriptArtifactFinding struct {
	Kind          string                       `json:"kind"`
	Recognizer    string                       `json:"recognizer"`
	OuterLocation string                       `json:"outer_location,omitempty"`
	Location      string                       `json:"location"`
	Artifact      *transcriptArtifactReference `json:"artifact,omitempty"`
	TextOffset    int                          `json:"text_offset"`
	Text          string                       `json:"text,omitempty"`
}
type transcriptArtifactReference struct {
	RecordedPath string `json:"recorded_path"`
	Location     string `json:"location"`
	Trust        string `json:"trust"`
}
type transcriptArtifactIssue struct {
	Location string `json:"location"`
	Reason   string `json:"reason"`
}
type diagnosticText struct{ text, location string }

func diagnoseTranscriptArtifactEvent(e ledgerEvent) transcriptArtifactDiagnostic {
	r := transcriptArtifactDiagnostic{Version: transcriptArtifactVersion, EventID: stringValue(e["event_id"]), Status: transcriptArtifactNotDetected, Findings: []transcriptArtifactFinding{}, Issues: []transcriptArtifactIssue{}, InspectionComplete: true}
	if stringValue(e["kind"]) != "tool_result" {
		return r
	}
	texts, complete, issues := diagnosticTexts(e["payload"])
	r.InspectionComplete = complete
	r.Issues = issues
	for _, t := range texts {
		fs, ok, is := diagnoseText(t.text, t.location)
		r.Issues = append(r.Issues, is...)
		if !ok {
			r.InspectionComplete = false
		}
		for _, f := range fs {
			if len(r.Findings) >= 16 {
				r.InspectionComplete = false
				r.Issues = append(r.Issues, transcriptArtifactIssue{t.location, "finding limit exceeded"})
				break
			}
			r.Findings = append(r.Findings, f)
		}
	}
	if len(r.Findings) > 0 {
		r.Status = transcriptArtifactDetected
	} else if !r.InspectionComplete {
		r.Status = transcriptArtifactUninspected
	}
	return r
}

func diagnosticTexts(v any) ([]diagnosticText, bool, []transcriptArtifactIssue) {
	issues := []transcriptArtifactIssue{}
	complete := true
	if raw, ok := v.(json.RawMessage); ok {
		if len(raw) > 4*1024*1024 {
			return nil, false, []transcriptArtifactIssue{{"/payload", "raw payload exceeds 4 MiB admission bound"}}
		}
		if d := jsonDepth(raw); d > 16 {
			return nil, false, []transcriptArtifactIssue{{"/payload", "JSON depth exceeds 16"}}
		}
		var x any
		if json.Unmarshal(raw, &x) != nil {
			return nil, false, []transcriptArtifactIssue{{"/payload", "malformed JSON payload"}}
		}
		v = x
	}
	out := []diagnosticText{}
	add := func(s, loc string) {
		if len(s) > 65536 {
			complete = false
			issues = append(issues, transcriptArtifactIssue{loc, "candidate text exceeds 64 KiB bound"})
			return
		}
		out = append(out, diagnosticText{s, loc})
	}
	switch x := v.(type) {
	case string:
		add(x, "/payload")
	case []any:
		for i, b := range x {
			if i >= 32 {
				complete = false
				issues = append(issues, transcriptArtifactIssue{"/payload", "content block limit exceeded"})
				break
			}
			if m, ok := b.(map[string]any); ok && m["type"] == "text" {
				if s, ok := m["text"].(string); ok {
					add(s, fmt.Sprintf("/payload/%d/text", i))
				}
			}
		}
	case map[string]any:
		if s, ok := x["content"].(string); ok {
			add(s, "/payload/content")
		}
		if bs, ok := x["content"].([]any); ok {
			for i, b := range bs {
				if i >= 32 {
					complete = false
					issues = append(issues, transcriptArtifactIssue{"/payload/content", "content block limit exceeded"})
					break
				}
				if m, ok := b.(map[string]any); ok && m["type"] == "text" {
					if s, ok := m["text"].(string); ok {
						add(s, fmt.Sprintf("/payload/content/%d/text", i))
					}
				}
			}
		}
	default:
		return nil, false, []transcriptArtifactIssue{{"/payload", "unsupported payload representation"}}
	}
	return out, complete, issues
}

func diagnoseText(s, outer string) ([]transcriptArtifactFinding, bool, []transcriptArtifactIssue) {
	out := []transcriptArtifactFinding{}
	issues := []transcriptArtifactIssue{}
	if i := strings.Index(s, "[MCP text output truncated:"); i >= 0 {
		if end := strings.Index(s[i:], "]"); end > len("[MCP text output truncated:") {
			out = append(out, transcriptArtifactFinding{Kind: transcriptArtifactTruncation, Recognizer: "mcp-text-truncation/v1", OuterLocation: outer, Location: outer, TextOffset: i, Text: s[i : i+end+1]})
		}
	}
	trim := strings.TrimSpace(s)
	if trim == "" || (trim[0] != '{' && trim[0] != '[') {
		return out, true, issues
	}
	if len(s) > 65536 {
		return out, false, []transcriptArtifactIssue{{outer, "candidate text exceeds 64 KiB bound"}}
	}
	if jsonDepth([]byte(s)) > 16 {
		return out, false, []transcriptArtifactIssue{{outer, "JSON depth exceeds 16"}}
	}
	var x any
	if json.Unmarshal([]byte(s), &x) != nil {
		return out, false, []transcriptArtifactIssue{{outer, "malformed JSON candidate"}}
	}
	m, ok := x.(map[string]any)
	if !ok {
		return out, true, issues
	}
	innerPrefix := ""
	if data, exists := m["data"].(map[string]any); exists {
		if _, isBool := m["ok"].(bool); isBool {
			m = data
			innerPrefix = "/data"
		}
	}
	if !boolValue(m["omitted"]) || stringValue(m["reason"]) == "" || !nonnegativeInt(m["contentBlocks"]) || !nonnegativeInt(m["rawResultBytes"]) {
		return out, true, issues
	}
	cs, ok := m["contentSummary"].([]any)
	if !ok {
		return out, true, issues
	}
	has := false
	for _, v := range cs {
		if mm, ok := v.(map[string]any); ok && boolValue(mm["textOmitted"]) {
			has = true
		}
	}
	if !has {
		return out, true, issues
	}
	f := transcriptArtifactFinding{Kind: transcriptArtifactElision, Recognizer: "adapter-elision/v1", OuterLocation: outer, Location: innerPrefix + "/omitted"}
	if p, ok := m["fullResultPath"].(string); ok && p != "" {
		f.Artifact = &transcriptArtifactReference{p, innerPrefix + "/fullResultPath", "recorded_unverified"}
	}
	return append(out, f), true, issues
}

func jsonDepth(b []byte) int {
	depth, max := 0, 0
	in, esc := false, false
	for _, c := range b {
		if in {
			if esc {
				esc = false
			} else if c == '\\' {
				esc = true
			} else if c == '"' {
				in = false
			}
			continue
		}
		if c == '"' {
			in = true
		} else if c == '{' || c == '[' {
			depth++
			if depth > max {
				max = depth
			}
		} else if c == '}' || c == ']' {
			depth--
		}
	}
	return max
}
func stringValue(v any) string  { s, _ := v.(string); return s }
func boolValue(v any) bool      { b, _ := v.(bool); return b }
func nonnegativeInt(v any) bool { n, ok := v.(float64); return ok && n >= 0 && n == float64(int64(n)) }
