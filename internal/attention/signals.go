package attention

// Builtins is deliberately a static collection, not a plugin registry.
func Builtins() ([]*Policy, error) {
	p, err := Builtin()
	if err != nil {
		return nil, err
	}
	return []*Policy{p}, nil
}

type Applicability struct {
	Status string `json:"status"`
	Task   string `json:"task,omitempty"`
	Source string `json:"source"`
	Reason string `json:"reason"`
}

type SignalResult struct {
	Applicability Applicability `json:"applicability"`
	Condition     Result        `json:"condition"`
	Outcome       string        `json:"outcome"`
	Error         string        `json:"error,omitempty"`
}

// Observe measures the condition even when task applicability is unknown. The
// historical Evaluate API remains unchanged for existing users and old captures.
func (p *Policy) Observe(thread, task, source string, m Metrics) SignalResult {
	a := Applicability{Status: "unknown", Source: "not_established", Reason: "Task classification not established; condition measured independently"}
	if task != "" {
		a = Applicability{Status: "not_applicable", Task: task, Source: source, Reason: "Established task is outside this signal's scope"}
		if task == p.Scope.Task {
			a.Status = "applicable"
			a.Reason = "Established task matches signal scope"
		}
	}
	condition, err := p.Evaluate(thread, p.Scope.Task, m)
	r := SignalResult{Applicability: a, Condition: condition}
	switch {
	case err != nil:
		r.Outcome = "error"
		r.Error = err.Error()
		r.Condition = Result{Status: "error", Reason: err.Error()}
	case a.Status == "not_applicable":
		r.Outcome = "not_applicable"
	case condition.Status == "indeterminate":
		r.Outcome = "insufficient_evidence"
	case a.Status == "unknown":
		r.Outcome = "needs_context"
	case condition.Status == "match":
		r.Outcome = "triggered"
	default:
		r.Outcome = "not_triggered"
	}
	return r
}
