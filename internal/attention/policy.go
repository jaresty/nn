// Package attention evaluates bundled transcript policies in isolated Datalog engines.
// It never loads notebook facts/rules, executes payloads, or accepts user policy files.
package attention

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/jaresty/nn/internal/rules"
	"gopkg.in/yaml.v3"
)

//go:embed builtin.yaml
var builtin string

func Builtin() (*Policy, error) { return parse(builtin) }

type Policy struct {
	ID      string `yaml:"id" json:"id"`
	Version int    `yaml:"version" json:"version"`
	Scope   struct {
		Task string `yaml:"task" json:"task"`
	} `yaml:"scope" json:"scope"`
	Window struct {
		LastWorkEvents int `yaml:"last_work_events" json:"last_work_events"`
	} `yaml:"window" json:"window"`
	Parameters map[string]float64 `yaml:"parameters" json:"parameters"`
	Rule       string             `yaml:"rule" json:"rule"`
	Digest     string             `yaml:"-" json:"digest"`
	parsed     rules.Rule
}

type Metrics struct {
	Commands   int  `json:"command_operations"`
	Edits      int  `json:"recognized_edit_operations"`
	Unknown    int  `json:"unknown_operations"`
	Neutral    int  `json:"neutral_operations"`
	Duplicates int  `json:"duplicate_invocations"`
	Rejected   int  `json:"validation_rejected_operations,omitempty"`
	Available  bool `json:"detail_available"`
}

type Result struct {
	Status string   `json:"status"`
	Reason string   `json:"reason"`
	Ratio  *float64 `json:"ratio"`
}

var identifier = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
var variable = regexp.MustCompile(`^[A-Z][a-zA-Z0-9_]*$`)

// parse only adds wrapper validation and AST admission checks. Rule syntax belongs
// exclusively to rules.ParseRule. One nonrecursive clause over singleton metric
// and parameter facts gives a statically bounded evaluation (no join explosion).
func parse(src string) (*Policy, error) {
	if len(src) > 8192 {
		return nil, fmt.Errorf("attention: definition exceeds 8192 bytes")
	}
	d := yaml.NewDecoder(strings.NewReader(src))
	d.KnownFields(true)
	var p Policy
	if e := d.Decode(&p); e != nil {
		return nil, e
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF {
		return nil, fmt.Errorf("attention: expected one YAML document")
	}
	if p.ID == "" || p.Version < 1 || p.Scope.Task == "" || p.Window.LastWorkEvents < 1 || p.Window.LastWorkEvents > 200 || len(p.Parameters) > 16 {
		return nil, fmt.Errorf("attention: invalid policy metadata or limits")
	}
	for k, v := range p.Parameters {
		if !identifier.MatchString(k) || math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
			return nil, fmt.Errorf("attention: invalid positive numeric parameter %q", k)
		}
	}
	r, e := rules.ParseRule(p.Rule)
	if e != nil {
		return nil, e
	}
	if r.Agg != nil || r.Head.Pred != "attention" || r.Head.Neg || len(r.Head.Args) != 2 || !r.Head.Args[0].Var || !variable.MatchString(r.Head.Args[0].Name) || r.Head.Args[1].Var || len(r.Head.Args[1].Name) > 160 || len(r.Body) == 0 || len(r.Body) > 16 {
		return nil, fmt.Errorf("attention: expected one bounded attention clause")
	}
	thread := r.Head.Args[0].Name
	numeric := map[string]bool{}
	threadBound := false
	comparisons := map[string]bool{"=cmp:lt": true, "=cmp:lte": true, "=cmp:gt": true, "=cmp:gte": true}
	for _, a := range r.Body {
		if a.Neg {
			return nil, fmt.Errorf("attention: negation is outside the admitted policy subset")
		}
		if comparisons[a.Pred] {
			if len(a.Args) != 2 {
				return nil, fmt.Errorf("attention: invalid comparison")
			}
			for _, term := range a.Args {
				if term.Var {
					if !numeric[term.Name] {
						return nil, fmt.Errorf("attention: numeric variable must be bound before comparison")
					}
				} else {
					v, e := strconv.ParseFloat(term.Name, 64)
					if e != nil || math.IsNaN(v) || math.IsInf(v, 0) {
						return nil, fmt.Errorf("attention: invalid numeric operand")
					}
				}
			}
			continue
		}
		arity := 2
		switch a.Pred {
		case "command_count", "recognized_edit_count", "edit_command_ratio":
		case "classification_complete":
			arity = 1
		case "parameter":
		default:
			return nil, fmt.Errorf("attention: unsupported predicate %q", a.Pred)
		}
		if len(a.Args) != arity {
			return nil, fmt.Errorf("attention: invalid arity for %s", a.Pred)
		}
		if a.Pred == "parameter" {
			if a.Args[0].Var {
				return nil, fmt.Errorf("attention: parameter key must be constant")
			}
			if _, ok := p.Parameters[a.Args[0].Name]; !ok {
				return nil, fmt.Errorf("attention: unknown parameter")
			}
		} else {
			if !a.Args[0].Var || a.Args[0].Name != thread {
				return nil, fmt.Errorf("attention: metric must bind head thread")
			}
			threadBound = true
		}
		if arity == 2 {
			v := a.Args[1]
			if !v.Var || !variable.MatchString(v.Name) || v.Name == thread {
				return nil, fmt.Errorf("attention: metric value must bind a numeric variable")
			}
			numeric[v.Name] = true
		}
	}
	if !threadBound {
		return nil, fmt.Errorf("attention: unbound thread")
	}
	p.parsed = r
	sum := sha256.Sum256([]byte(src))
	p.Digest = hex.EncodeToString(sum[:])
	return &p, nil
}

func (p *Policy) Evaluate(thread, task string, m Metrics) (Result, error) {
	result := Result{Status: "no_match", Reason: "Rule did not match; this is not a health assessment"}
	if m.Commands < 0 || m.Edits < 0 || m.Unknown < 0 || m.Neutral < 0 || m.Duplicates < 0 || m.Rejected < 0 || m.Commands+m.Edits+m.Unknown+m.Neutral+m.Duplicates+m.Rejected > 2000 {
		return Result{}, fmt.Errorf("attention: metric evaluation limit exceeded or invalid counts")
	}
	if task != p.Scope.Task {
		result.Status = "inapplicable"
		result.Reason = "Explicit task scope is missing or does not match policy"
		return result, nil
	}
	if !m.Available || m.Unknown > 0 || m.Commands == 0 {
		result.Status = "indeterminate"
		result.Reason = "Unavailable detail, unknown classification, or zero denominator"
		return result, nil
	}
	ratio := float64(m.Edits) / float64(m.Commands)
	result.Ratio = &ratio
	e := rules.NewEngine()
	add := func(pred string, args ...string) { e.AddFact(rules.Fact{Pred: pred, Args: args}) }
	add("command_count", thread, strconv.Itoa(m.Commands))
	add("recognized_edit_count", thread, strconv.Itoa(m.Edits))
	add("edit_command_ratio", thread, strconv.FormatFloat(ratio, 'g', -1, 64))
	add("classification_complete", thread)
	for k, v := range p.Parameters {
		add("parameter", k, strconv.FormatFloat(v, 'g', -1, 64))
	}
	e.AddRules([]rules.Rule{p.parsed})
	if err := e.Eval(); err != nil {
		return Result{}, err
	}
	matches := e.Query("attention")
	if len(matches) > 0 {
		result.Status = "match"
		result.Reason = matches[0].Args[1]
	}
	return result, nil
}
