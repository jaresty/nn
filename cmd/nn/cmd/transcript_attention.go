package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jaresty/nn/internal/attention"
	"github.com/spf13/cobra"
)

const attentionLimitations = "Recognized tool invocations, not successful edits or filesystem changes. Shell side effects are opaque. Unknown tools/arguments make classification indeterminate. Canonical owned-message window, not elapsed time. No liveness or health inference."

type attentionRoom struct {
	ID      string            `json:"agent_id"`
	Label   string            `json:"label"`
	Metrics attention.Metrics `json:"metrics"`
	Window  attentionWindow   `json:"window"`
	Result  attention.Result  `json:"result"`
}

type attentionPage struct {
	Version     int               `json:"version"`
	Snapshot    string            `json:"snapshot"`
	Path        string            `json:"path"`
	Schema      string            `json:"schema"`
	CaptureID   string            `json:"capture_id,omitempty"`
	CaptureMode string            `json:"capture_mode"`
	Task        string            `json:"task"`
	Policy      *attention.Policy `json:"policy"`
	Population  int               `json:"population"`
	Evaluated   int               `json:"evaluated"`
	Unevaluated int               `json:"unevaluated"`
	Limitations string            `json:"limitations"`
	Rooms       []attentionRoom   `json:"rooms"`
}

type attentionRetention struct {
	InputPath string                         `json:"input_path"`
	Page      attentionPage                  `json:"page"`
	Evidence  map[string][]attentionEvidence `json:"evidence"`
}

func newTranscriptAttentionCmd() *cobra.Command {
	var ids []string
	var task, snapshot, format string
	c := &cobra.Command{Use: "attention <session>", Short: "Evaluate the bundled attention policy for explicitly selected rooms (Pi and Claude)", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
		if format != "json" && format != "text" {
			return fmt.Errorf("attention: format must be json or text")
		}
		var retained attentionRetention
		var e error
		if snapshot != "" {
			retained, e = loadAttention(snapshot)
			if e != nil {
				return e
			}
			path, e := filepath.Abs(args[0])
			if e != nil {
				return e
			}
			if path != retained.InputPath && path != retained.Page.Path {
				return fmt.Errorf("attention: snapshot path mismatch")
			}
			if c.Flags().Changed("task") && task != retained.Page.Task {
				return fmt.Errorf("attention: snapshot task mismatch")
			}
			saved := []string{}
			for _, r := range retained.Page.Rooms {
				saved = append(saved, r.ID)
			}
			if c.Flags().Changed("agent") && !slices.Equal(ids, saved) {
				return fmt.Errorf("attention: snapshot selection mismatch")
			}
		} else {
			retained, e = buildAttention(args[0], ids, task)
			if e != nil {
				return e
			}
		}
		if format == "json" {
			return attentionJSON(c.OutOrStdout(), retained.Page)
		}
		return renderAttention(c.OutOrStdout(), retained.Page)
	}}
	c.Flags().StringArrayVar(&ids, "agent", nil, "exact room ID; repeat for up to 20 explicitly selected rooms, including ROOT")
	c.Flags().StringVar(&task, "task", "", "explicit task classification (implementation applies; absent/other is inapplicable)")
	c.Flags().StringVar(&snapshot, "snapshot", "", "replay a retained evaluation without source reads; optional selectors must match")
	c.Flags().StringVar(&format, "format", "text", "text or json")
	c.AddCommand(newAttentionInspectCmd())
	return c
}

func newAttentionInspectCmd() *cobra.Command {
	var page int
	var id, format string
	c := &cobra.Command{Use: "inspect <snapshot>", Short: "Inspect bounded retained work excerpts for one evaluated room", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
		if format != "json" && format != "text" {
			return fmt.Errorf("attention: format must be json or text")
		}
		r, e := loadAttention(args[0])
		if e != nil {
			return e
		}
		evidence, ok := r.Evidence[id]
		if !ok {
			return fmt.Errorf("attention: room was not evaluated in this snapshot")
		}
		total := len(evidence)
		pages := (total + 19) / 20
		if pages == 0 {
			pages = 1
		}
		if page < 1 || page > pages {
			return fmt.Errorf("attention: inspection page out of range")
		}
		start := (page - 1) * 20
		end := start + 20
		if end > total {
			end = total
		}
		evidence = evidence[start:end]
		var room attentionRoom
		for _, v := range r.Page.Rooms {
			if v.ID == id {
				room = v
				break
			}
		}
		if format == "json" {
			return attentionJSON(c.OutOrStdout(), struct {
				Snapshot   string              `json:"snapshot"`
				Room       attentionRoom       `json:"room"`
				Evidence   []attentionEvidence `json:"evidence"`
				Disclosure string              `json:"disclosure"`
				Page       int                 `json:"page"`
				Pages      int                 `json:"pages"`
				Total      int                 `json:"total_evidence_records"`
			}{args[0], room, evidence, "Bounded excerpts, not complete payloads; thinking omitted", page, pages, total})
		}
		var b bytes.Buffer
		p := r.Page
		p.Rooms = []attentionRoom{room}
		p.Evaluated = 1
		p.Unevaluated = p.Population - 1
		if e = renderAttention(&b, p); e != nil {
			return e
		}
		fmt.Fprintf(&b, "\nRetained evidence excerpts: page %d/%d · %d of %d work records (320 characters per record; thinking omitted)\n", page, pages, len(evidence), total)
		if page < pages {
			fmt.Fprintf(&b, "Next evidence page: --page %d\n", page+1)
		}
		for _, v := range evidence {
			fmt.Fprintf(&b, "\n%s · %s:%d · %s\n%s\n", v.EventID, cleanBundleText(v.Path), v.Ordinal, v.Role, v.Summary)
		}
		return attentionText(c.OutOrStdout(), b.String())
	}}
	c.Flags().IntVar(&page, "page", 1, "retained evidence page (20 work records per page)")
	c.Flags().StringVar(&id, "agent", "", "exact evaluated room ID")
	c.Flags().StringVar(&format, "format", "text", "text or json")
	return c
}

func buildAttention(session string, ids []string, task string) (attentionRetention, error) {
	var empty attentionRetention
	if len(ids) < 1 || len(ids) > 20 {
		return empty, fmt.Errorf("attention: select 1–20 exact --agent IDs; no implicit office-wide scan")
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || seen[id] {
			return empty, fmt.Errorf("attention: empty or duplicate agent ID")
		}
		seen[id] = true
	}
	if len(task) > 80 {
		return empty, fmt.Errorf("attention: task scope exceeds 80 bytes")
	}
	policy, e := attention.Builtin()
	if e != nil {
		return empty, e
	}
	input, e := filepath.Abs(session)
	if e != nil {
		return empty, e
	}
	path, e := filepath.EvalSymlinks(input)
	if e != nil {
		return empty, e
	}
	schema := classifyTranscript(path)
	var agents []agent
	var capture *transcriptCapture
	if schema == schemaPi {
		capture, e = newTranscriptCaptureForAgents(path, seen)
		if e != nil {
			return empty, e
		}
		agents, e = buildPiTreeUsing(capture.Path, capture.read, capture.resolve)
	} else {
		agents, e = buildTree(path)
	}
	if e != nil {
		return empty, e
	}
	byID := map[string]agent{}
	for _, a := range agents {
		byID[a.ID] = a
	}
	for _, id := range ids {
		if _, ok := byID[id]; !ok {
			return empty, fmt.Errorf("attention: unknown agent %q", id)
		}
	}
	p := attentionPage{Version: 1, Path: path, Schema: schema, Task: task, Policy: policy, Population: len(agents), Evaluated: len(ids), Unevaluated: len(agents) - len(ids), Limitations: attentionLimitations, CaptureMode: "sequential retained owned-record projections", Rooms: []attentionRoom{}}
	if capture != nil {
		p.CaptureID = capture.ID
		p.CaptureMode = "root plus selected sidechain complete-record prefixes"
	}
	retained := attentionRetention{InputPath: input, Evidence: map[string][]attentionEvidence{}}
	for _, id := range ids {
		var records []ledgerRecord
		var status string
		if capture != nil {
			records, status = capture.ledger(id)
		} else {
			records, _, status, e = ledgerRecords(path, id)
			if e != nil {
				return empty, e
			}
		}
		metrics, window, evidence, e := collectAttention(records, status, id, policy.Window.LastWorkEvents)
		if e != nil {
			return empty, e
		}
		result, e := policy.Evaluate(id, task, metrics)
		if e != nil {
			return empty, e
		}
		label := byID[id].Description
		if label == "" {
			label = "Room " + id
		}
		label = reviewLabel(cleanBundleText(label))
		p.Rooms = append(p.Rooms, attentionRoom{ID: id, Label: label, Metrics: metrics, Window: window, Result: result})
		retained.Evidence[id] = evidence
	}
	retained.Page = p
	b, e := json.Marshal(retained)
	if e != nil {
		return empty, e
	}
	if len(b) > 4*1024*1024 {
		return empty, fmt.Errorf("attention: retained evidence exceeds 4 MiB")
	}
	snapshot := captureHash(b)
	cleanupTranscriptCaptures()
	if e = writeCaptureFile(snapshot+".attention", b); e != nil {
		return empty, e
	}
	retained.Page.Snapshot = snapshot
	return retained, nil
}

func loadAttention(snapshot string) (attentionRetention, error) {
	var r attentionRetention
	if !validCaptureID(snapshot) {
		return r, fmt.Errorf("attention: invalid snapshot")
	}
	b, e := readCaptureFile(snapshot + ".attention")
	if e != nil {
		return r, fmt.Errorf("attention: retained evaluation unavailable: %w", e)
	}
	if captureHash(b) != snapshot || json.Unmarshal(b, &r) != nil || r.Page.Version != 1 {
		return attentionRetention{}, fmt.Errorf("attention: corrupt retained evaluation")
	}
	r.Page.Snapshot = snapshot
	return r, nil
}

func attentionJSON(w io.Writer, v any) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	if len(b) > 200000 {
		return fmt.Errorf("attention: JSON output exceeds 200000 bytes")
	}
	_, e = w.Write(append(b, '\n'))
	return e
}
func attentionText(w io.Writer, text string) error {
	if len([]rune(text)) > 100000 {
		return fmt.Errorf("attention: text output exceeds 100000 characters")
	}
	_, e := io.WriteString(w, text)
	return e
}
func renderAttention(w io.Writer, p attentionPage) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Attention signals · %d evaluated · %d unevaluated in selected transcript\nSnapshot: %s\nPolicy: %s v%d · %s\nTask: %s\nCapture: %s\n", p.Evaluated, p.Unevaluated, p.Snapshot, p.Policy.ID, p.Policy.Version, p.Policy.Digest, cleanBundleText(p.Task), p.CaptureMode)
	fmt.Fprintf(&b, "Parameters: %v\n", p.Policy.Parameters)
	for _, r := range p.Rooms {
		ratio := "undefined"
		if r.Result.Ratio != nil {
			ratio = fmt.Sprintf("%.5g", *r.Result.Ratio)
		}
		fmt.Fprintf(&b, "\n%s · %s\n%s — %s\n%d recognized edits / %d commands = %s; unknown=%d\nWindow: %d/%d work records; %d earlier ledger records omitted; unknown timestamps=%d\nInspect evidence: nn transcript attention inspect %s --agent %s\n", r.Label, cleanBundleText(r.ID), r.Result.Status, r.Result.Reason, r.Metrics.Edits, r.Metrics.Commands, ratio, r.Metrics.Unknown, r.Window.Selected, r.Window.Requested, r.Window.Earlier, r.Window.UnknownTimestamps, p.Snapshot, "'"+strings.ReplaceAll(r.ID, "'", "'\\''")+"'")
	}
	fmt.Fprintf(&b, "\n%s\n", p.Limitations)
	return attentionText(w, b.String())
}
