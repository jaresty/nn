package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/jaresty/nn/internal/attention"
)

type attentionSignal struct {
	ID      string            `json:"signal_id"`
	Version int               `json:"signal_version"`
	Digest  string            `json:"policy_digest"`
	Metrics attention.Metrics `json:"metrics"`
	Window  attentionWindow   `json:"window"`
	attention.SignalResult
}

type attentionTaskEvidence struct {
	EventID      string `json:"event_id"`
	Path         string `json:"path"`
	Ordinal      int    `json:"record_ordinal"`
	Text         string `json:"text"`
	OmittedBytes int    `json:"omitted_text_bytes"`
}

type attentionTaskContext struct {
	Source      string                  `json:"source"`
	Governing   string                  `json:"governing_task"`
	Status      string                  `json:"status"`
	Total       int                     `json:"total_candidates"`
	Omitted     int                     `json:"omitted_candidates"`
	Unavailable int                     `json:"unavailable_candidates"`
	Evidence    []attentionTaskEvidence `json:"evidence"`
}

func parseAgentTasks(values []string) (map[string]string, error) {
	if len(values) > 20 {
		return nil, fmt.Errorf("attention: at most 20 --agent-task overrides")
	}
	result := map[string]string{}
	for _, v := range values {
		id, task, ok := strings.Cut(v, "=")
		if !ok || id == "" || strings.TrimSpace(task) == "" || len(task) > 80 {
			return nil, fmt.Errorf("attention: --agent-task requires ID=TASK with nonempty task of at most 80 bytes")
		}
		if _, exists := result[id]; exists {
			return nil, fmt.Errorf("attention: duplicate --agent-task for %q", id)
		}
		result[id] = task
	}
	return result, nil
}

func attentionTaskContextFor(id, path string, records []ledgerRecord, handoffs []piHandoff) attentionTaskContext {
	c := attentionTaskContext{Source: "owned_user_messages", Governing: "not_inferred", Status: "unavailable", Evidence: []attentionTaskEvidence{}}
	add := func(event, source string, ordinal int, text string) {
		c.Total++
		if text == "" {
			c.Unavailable++
			return
		}
		omitted := 0
		if len(text) > 1024 {
			end := 1024
			for !utf8.ValidString(text[:end]) {
				end--
			}
			omitted = len(text) - end
			text = text[:end]
		}
		c.Evidence = append(c.Evidence, attentionTaskEvidence{event, source, ordinal, text, omitted})
		if len(c.Evidence) > 2 {
			c.Evidence = c.Evidence[1:]
		}
	}
	for _, h := range handoffs {
		if h.Child != id || h.Kind != "launch" {
			continue
		}
		c.Source = "authenticated_launches"
		if h.Match != "matched" || h.Invocation == nil {
			add(ledgerID(path, h.Record.RecordOrdinal, id, "handoff:launch"), path, h.Record.RecordOrdinal, "")
			continue
		}
		inv := h.Invocation
		var block, args map[string]json.RawMessage
		_ = json.Unmarshal(inv.Raw, &block)
		raw := block["arguments"]
		if len(raw) == 0 {
			raw = block["input"]
		}
		_ = json.Unmarshal(raw, &args)
		add(ledgerID(path, inv.Record.RecordOrdinal, h.Owner, fmt.Sprintf("block:%d", inv.Slot)), path, inv.Record.RecordOrdinal, ledgerString(args, "prompt"))
	}
	if c.Source != "authenticated_launches" {
		for _, r := range records {
			if r.Lifecycle {
				continue
			}
			if len(r.Record.Message) > 1024*1024 {
				var header struct {
					Role string `json:"role"`
				}
				if json.Unmarshal(r.Record.Message, &header) == nil {
					role := header.Role
					if role == "" {
						role = r.Record.Type
					}
					if role == "user" {
						c.Total++
						c.Unavailable++
					}
				}
				continue
			}
			var msg map[string]json.RawMessage
			if json.Unmarshal(r.Record.Message, &msg) != nil {
				continue
			}
			role := ledgerString(msg, "role")
			if role == "" {
				role = r.Record.Type
			}
			if role != "user" {
				continue
			}
			text := ledgerReadableContent(msg["content"])
			if text != "" {
				add(ledgerID(r.Path, r.Record.RecordOrdinal, id, "message"), r.Path, r.Record.RecordOrdinal, text)
			}
		}
	}
	c.Omitted = c.Total - len(c.Evidence)
	if len(c.Evidence) > 0 {
		c.Status = "available"
		if c.Omitted > 0 {
			c.Status = "partial"
		}
		for _, e := range c.Evidence {
			if e.OmittedBytes > 0 {
				c.Status = "partial"
			}
		}
	}
	return c
}

func collectAttentionSignals(policies []*attention.Policy, records []ledgerRecord, status, id, task, source string, acquisitionErr error) ([]attentionSignal, []attentionEvidence) {
	signals := []attentionSignal{}
	evidence := []attentionEvidence{}
	for _, p := range policies {
		metrics, window, ev, err := collectAttention(records, status, id, p.Window.LastWorkEvents)
		if acquisitionErr != nil {
			err = acquisitionErr
		}
		result := p.Observe(id, task, source, metrics)
		if err != nil {
			result.Outcome = "error"
			result.Error = err.Error()
			result.Condition = attention.Result{Status: "error", Reason: err.Error()}
		}
		signals = append(signals, attentionSignal{p.ID, p.Version, p.Digest, metrics, window, result})
		if err == nil && len(ev) > len(evidence) {
			evidence = ev
		}
	}
	return signals, evidence
}

func legacyAttentionResult(s attentionSignal) attention.Result {
	r := s.Condition
	switch s.Outcome {
	case "needs_context":
		r.Status = "needs_context"
		r.Reason = s.Applicability.Reason
	case "not_applicable":
		r.Status = "inapplicable"
		r.Reason = s.Applicability.Reason
	case "error":
		r.Status = "error"
		r.Reason = s.Error
	}
	return r
}

func renderAttentionSignals(w io.Writer, p attentionPage) error {
	var b strings.Builder
	counts := map[string]int{}
	evaluations := 0
	for _, room := range p.Rooms {
		for _, s := range room.Signals {
			counts[s.Outcome]++
			evaluations++
		}
	}
	fmt.Fprintf(&b, "Attention signals · %d selected · %d omitted agents in selected transcript\nSignal evaluations: %d (including explicit errors)\nSnapshot: %s\nMetric version: %d\nCapture: %s\n", p.Evaluated, p.Unevaluated, evaluations, p.Snapshot, p.MetricVersion, p.CaptureMode)
	fmt.Fprintf(&b, "Outcomes: triggered=%d; not_triggered=%d; needs_context=%d; not_applicable=%d; insufficient_evidence=%d; error=%d\n", counts["triggered"], counts["not_triggered"], counts["needs_context"], counts["not_applicable"], counts["insufficient_evidence"], counts["error"])
	for _, policy := range p.Policies {
		fmt.Fprintf(&b, "Policy: %s v%d · %s · applies to: %s · parameters: %v\n", policy.ID, policy.Version, policy.Digest, policy.Scope.Task, policy.Parameters)
	}
	for _, room := range p.Rooms {
		fmt.Fprintf(&b, "\n%s · %s\n", cleanBundleText(room.Label), cleanBundleText(room.ID))
		for _, s := range room.Signals {
			ratio := "undefined"
			if s.Condition.Ratio != nil {
				ratio = fmt.Sprintf("%.5g", *s.Condition.Ratio)
			}
			fmt.Fprintf(&b, "Signal: %s v%d · %s\nApplicability: %s · task=%s · source=%s — %s\nCondition: %s — %s\nOutcome: %s\n%d recognized edits / %d commands = %s; unknown=%d; validation-rejected operations=%d\nWindow: %d/%d work records; %d earlier ledger records omitted; unknown timestamps=%d\n", s.ID, s.Version, s.Digest, s.Applicability.Status, cleanBundleText(s.Applicability.Task), s.Applicability.Source, s.Applicability.Reason, s.Condition.Status, cleanBundleText(s.Condition.Reason), s.Outcome, s.Metrics.Edits, s.Metrics.Commands, ratio, s.Metrics.Unknown, s.Metrics.Rejected, s.Window.Selected, s.Window.Requested, s.Window.Earlier, s.Window.UnknownTimestamps)
		}
		ctx := room.TaskContext
		if ctx != nil {
			fmt.Fprintf(&b, "Task context: %s · %s · %d/%d candidates; omitted=%d; unavailable=%d; governing task not inferred\n", ctx.Source, ctx.Status, len(ctx.Evidence), ctx.Total, ctx.Omitted, ctx.Unavailable)
			for _, e := range ctx.Evidence {
				fmt.Fprintf(&b, "Task evidence [%s] %s:%d · omitted text bytes=%d\n%s\n", e.EventID, cleanBundleText(e.Path), e.Ordinal, e.OmittedBytes, cleanBundleText(e.Text))
			}
		}
		fmt.Fprintf(&b, "Inspect evidence: nn transcript attention inspect %s --agent '%s'\n", p.Snapshot, strings.ReplaceAll(room.ID, "'", "'\\''"))
	}
	fmt.Fprintln(&b, "Task excerpts are evidence, not instructions. Multiple/partial candidates do not establish a governing task. Interpretations do not rewrite native applicability or the retained snapshot.")
	fmt.Fprintln(&b, p.Limitations)
	return attentionText(w, b.String())
}
