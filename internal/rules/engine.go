// Package rules implements a small pure-Go semi-naive Datalog engine that
// evaluates rules over facts derived from notes. It is a pure derivation layer:
// facts and rules both come from the Markdown on disk, and the engine never
// mutates truth — it only computes derived facts (per ADR-0019).
package rules

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ── Terms, atoms, rules, facts ────────────────────────────────────────────────

// Term is either a variable (Var true) or a constant (Var false).
type Term struct {
	Name string
	Var  bool
}

// Atom is a predicate applied to terms. Neg marks a negated body literal.
type Atom struct {
	Pred string
	Args []Term
	Neg  bool
}

// Rule is a head atom derived from a conjunction of body atoms. An empty Body
// makes the rule a ground fact assertion. When Agg is non-nil the rule is an
// aggregate rule (see aggregate) rather than an ordinary join.
type Rule struct {
	Head Atom
	Body []Atom
	Agg  *aggregate
}

// aggregate describes a count rule of the form
//
//	head(G..., K) :- count(V : source(...)) = K.
//
// where source is a single literal, V is the counted variable, and the head's
// arguments other than the result variable are the grouping variables. K is
// bound to the number of DISTINCT V values within each group.
type aggregate struct {
	countVar string // the variable counted (V)
	source   Atom   // the single source literal
	resultOn int    // index into Head.Args that receives the count (K)
}

// Fact is a ground atom: a predicate with constant string arguments.
type Fact struct {
	Pred string
	Args []string
}

// Key returns a canonical string identity for a fact.
func (f Fact) Key() string {
	return f.Pred + "(" + strings.Join(f.Args, ",") + ")"
}

// ── Engine ─────────────────────────────────────────────────────────────────────

// Engine holds a fact base and a ruleset and computes their least fixpoint.
type Engine struct {
	facts map[string]Fact
	rules []Rule
	// deriv records, for each derived fact key, one witness of how it was
	// produced: the rule that fired and the premise fact keys it consumed. Base
	// facts (added via AddFact) are absent from this map.
	deriv map[string]derivation
}

// derivation is one witness of how a derived fact was produced.
type derivation struct {
	rule     Rule
	premises []string // fact keys
}

// NewEngine returns an empty engine.
func NewEngine() *Engine {
	return &Engine{facts: map[string]Fact{}, deriv: map[string]derivation{}}
}

// AddFact inserts a ground fact into the fact base.
func (e *Engine) AddFact(f Fact) { e.facts[f.Key()] = f }

// AddRules appends rules to the ruleset.
func (e *Engine) AddRules(rs []Rule) { e.rules = append(e.rules, rs...) }

// has reports whether a ground fact is present.
func (e *Engine) has(f Fact) bool { _, ok := e.facts[f.Key()]; return ok }

// Query returns all facts for the given predicate, sorted by key.
func (e *Engine) Query(pred string) []Fact {
	var out []Fact
	for _, f := range e.facts {
		if f.Pred == pred {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out
}

// Eval validates safety and stratification, then runs the least-fixpoint
// computation. It returns an error if a rule is unsafe (a comparison operand is
// never bound by a positive literal) or the ruleset is not stratifiable.
func (e *Engine) Eval() error {
	if err := checkSafe(e.rules); err != nil {
		return err
	}
	if err := checkStratified(e.rules); err != nil {
		return err
	}
	// Interleave ordinary fixpoint and aggregate evaluation until neither adds a
	// fact. Aggregates read the completed source relation each round; downstream
	// rules then see the aggregate's output on the next fixpoint pass.
	for {
		e.fixpoint()
		if !e.evalAggregates() {
			return nil
		}
	}
}

// evalAggregates runs every aggregate rule against the current facts, emitting
// count facts. It returns whether any new fact was added.
func (e *Engine) evalAggregates() bool {
	added := false
	for _, r := range e.rules {
		if r.Agg == nil {
			continue
		}
		for _, f := range e.aggregateFacts(r) {
			if !e.has(f) {
				e.facts[f.Key()] = f
				added = true
			}
		}
	}
	return added
}

// aggregateFacts computes the head facts produced by an aggregate rule: for each
// group (distinct binding of the head's non-result variables), count the DISTINCT
// values of the counted variable among matching source facts.
func (e *Engine) aggregateFacts(r Rule) []Fact {
	agg := r.Agg
	// group key (ordered head arg values excluding the result slot) -> set of
	// distinct counted values.
	groups := map[string]map[string]bool{}
	groupArgs := map[string][]string{}

	for _, f := range e.facts {
		b, ok := unify(agg.source, f, bindings{})
		if !ok {
			continue
		}
		countVal, bound := b[agg.countVar]
		if !bound {
			continue
		}
		// Build the group's head-argument vector (result slot left empty).
		args := make([]string, len(r.Head.Args))
		ok = true
		for i, t := range r.Head.Args {
			if i == agg.resultOn {
				continue
			}
			if t.Var {
				v, has := b[t.Name]
				if !has {
					ok = false
					break
				}
				args[i] = v
			} else {
				args[i] = t.Name
			}
		}
		if !ok {
			continue
		}
		key := strings.Join(args, "\x00")
		if groups[key] == nil {
			groups[key] = map[string]bool{}
			groupArgs[key] = args
		}
		groups[key][countVal] = true
	}

	var out []Fact
	for key, distinct := range groups {
		args := append([]string(nil), groupArgs[key]...)
		args[agg.resultOn] = strconv.Itoa(len(distinct))
		out = append(out, Fact{Pred: r.Head.Pred, Args: args})
	}
	return out
}

// checkSafe rejects rules where a comparison literal references a variable that
// is not bound by any positive (non-comparison, non-negated) body literal —
// such a comparison is not range-restricted and cannot be evaluated.
func checkSafe(rules []Rule) error {
	for _, r := range rules {
		bound := map[string]bool{}
		for _, lit := range r.Body {
			if lit.Neg || lit.Pred == predEq || lit.Pred == predNeq {
				continue
			}
			for _, t := range lit.Args {
				if t.Var && t.Name != "_" {
					bound[t.Name] = true
				}
			}
		}
		for _, lit := range r.Body {
			if lit.Pred != predEq && lit.Pred != predNeq {
				continue
			}
			for _, t := range lit.Args {
				if t.Var && t.Name != "_" && !bound[t.Name] {
					return fmt.Errorf("rules: unsafe comparison — variable %q is not bound by any positive literal", t.Name)
				}
			}
		}
	}
	return nil
}

// fixpoint iterates the rules until no new fact is derived.
func (e *Engine) fixpoint() {
	for {
		added := false
		for _, r := range e.rules {
			if r.Agg != nil {
				// Aggregate rules are evaluated by evalAggregates, not the
				// ordinary join pass; their empty Body would otherwise emit a
				// spurious all-empty-args head fact.
				continue
			}
			for _, b := range e.evalBody(r.Body) {
				nf := subst(r.Head, b)
				if !e.has(nf) {
					e.facts[nf.Key()] = nf
					e.deriv[nf.Key()] = derivation{rule: r, premises: positivePremiseKeys(r.Body, b)}
					added = true
				}
			}
		}
		if !added {
			return
		}
	}
}

type bindings map[string]string

// evalBody returns every variable binding under which the whole body holds.
// A negated literal is satisfied (as failure) when no matching fact exists once
// its variables are bound by earlier literals.
func (e *Engine) evalBody(body []Atom) []bindings {
	results := []bindings{{}}
	for _, lit := range body {
		var next []bindings
		for _, b := range results {
			switch {
			case lit.Pred == predEq || lit.Pred == predNeq:
				if compareHolds(lit, b) {
					next = append(next, b)
				}
			case lit.Neg:
				if !e.has(subst(lit, b)) {
					next = append(next, b)
				}
			default:
				for _, f := range e.facts {
					if nb, ok := unify(lit, f, b); ok {
						next = append(next, nb)
					}
				}
			}
		}
		results = next
	}
	return results
}

// compareHolds evaluates a comparison literal (predEq/predNeq) under bindings b.
// Both operands must be bound to a constant (variable resolved via b, or a
// literal constant); an unbound operand is treated as a failed comparison.
func compareHolds(lit Atom, b bindings) bool {
	l, lok := resolveTerm(lit.Args[0], b)
	r, rok := resolveTerm(lit.Args[1], b)
	if !lok || !rok {
		return false
	}
	if lit.Pred == predEq {
		return l == r
	}
	return l != r
}

// resolveTerm returns the constant value a term denotes under bindings b, and
// whether it is bound (constants are always bound; a wildcard is never bound).
func resolveTerm(t Term, b bindings) (string, bool) {
	if !t.Var {
		return t.Name, true
	}
	if t.Name == "_" {
		return "", false
	}
	v, ok := b[t.Name]
	return v, ok
}

// unify attempts to match atom a against fact f under existing bindings b,
// returning the extended bindings on success.
func unify(a Atom, f Fact, b bindings) (bindings, bool) {
	if a.Pred != f.Pred || len(a.Args) != len(f.Args) {
		return nil, false
	}
	nb := make(bindings, len(b)+len(a.Args))
	for k, v := range b {
		nb[k] = v
	}
	for i, t := range a.Args {
		switch {
		case t.Var && t.Name == "_":
			// wildcard: matches anything, binds nothing
		case t.Var:
			if cur, ok := nb[t.Name]; ok {
				if cur != f.Args[i] {
					return nil, false
				}
			} else {
				nb[t.Name] = f.Args[i]
			}
		default:
			if t.Name != f.Args[i] {
				return nil, false
			}
		}
	}
	return nb, true
}

// subst grounds an atom under bindings, producing a fact.
func subst(a Atom, b bindings) Fact {
	f := Fact{Pred: a.Pred}
	for _, t := range a.Args {
		if t.Var {
			f.Args = append(f.Args, b[t.Name])
		} else {
			f.Args = append(f.Args, t.Name)
		}
	}
	return f
}

// ── Stratification ─────────────────────────────────────────────────────────────

// checkStratified rejects rulesets where a predicate depends, through a cycle,
// on the negation of a predicate — the classic non-stratifiable case that makes
// the least fixpoint ill-defined. It builds a dependency graph whose edges are
// labelled positive/negative and reports an error if any strongly-connected
// cycle contains a negative edge.
func checkStratified(rules []Rule) error {
	// negEdge[a][b] = true means "a depends negatively on b".
	// posEdge[a] lists all (pos or neg) dependencies of a.
	deps := map[string][]dep{}
	for _, r := range rules {
		head := r.Head.Pred
		if r.Agg != nil {
			// An aggregate reads a completed source relation; model the
			// source dependency as NEGATIVE so a source that cycles back to
			// this aggregate's head is rejected as a negative cycle (the count
			// would otherwise never converge).
			deps[head] = append(deps[head], dep{pred: r.Agg.source.Pred, neg: true})
			continue
		}
		for _, lit := range r.Body {
			// Comparison literals are not predicate dependencies.
			if lit.Pred == predEq || lit.Pred == predNeq {
				continue
			}
			deps[head] = append(deps[head], dep{pred: lit.Pred, neg: lit.Neg})
		}
	}

	// Detect a cycle that traverses at least one negative edge via DFS,
	// tracking whether the current path has crossed a negative edge.
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := map[string]int{}
	var visit func(pred string, negOnPath bool) error
	visit = func(pred string, negOnPath bool) error {
		color[pred] = gray
		for _, d := range deps[pred] {
			pathNeg := negOnPath || d.neg
			if d.pred == pred && d.neg {
				return fmt.Errorf("rules: predicate %q negatively depends on itself (not stratifiable)", pred)
			}
			switch color[d.pred] {
			case gray:
				if pathNeg {
					return fmt.Errorf("rules: predicate %q participates in a negative cycle (not stratifiable)", d.pred)
				}
			case white:
				if err := visit(d.pred, pathNeg); err != nil {
					return err
				}
			}
		}
		color[pred] = black
		return nil
	}

	for pred := range deps {
		if color[pred] == white {
			if err := visit(pred, false); err != nil {
				return err
			}
		}
	}
	return nil
}

type dep struct {
	pred string
	neg  bool
}

// ── Provenance / explanation ───────────────────────────────────────────────────

// positivePremiseKeys returns the grounded fact keys of the positive body
// literals under bindings b — the facts a rule firing consumed. Negated literals
// are not premises (they assert absence) and are skipped.
func positivePremiseKeys(body []Atom, b bindings) []string {
	var keys []string
	for _, lit := range body {
		if lit.Neg {
			continue
		}
		keys = append(keys, subst(lit, b).Key())
	}
	return keys
}

// Explain returns a human-readable, recursively-expanded derivation of a fact:
// the rule that produced it and, beneath, the derivation of each premise, down
// to base facts. ok is false if the fact was never established.
func (e *Engine) Explain(f Fact) (steps []string, ok bool) {
	if !e.has(f) {
		return nil, false
	}
	seen := map[string]bool{}
	e.explain(f.Key(), 0, seen, &steps)
	return steps, true
}

func (e *Engine) explain(key string, depth int, seen map[string]bool, out *[]string) {
	indent := strings.Repeat("  ", depth)
	if seen[key] {
		*out = append(*out, indent+key+" (already shown)")
		return
	}
	seen[key] = true

	d, derived := e.deriv[key]
	if !derived {
		*out = append(*out, indent+key+" — base fact")
		return
	}
	*out = append(*out, indent+key+" — via rule: "+d.rule.String())
	for _, p := range d.premises {
		e.explain(p, depth+1, seen, out)
	}
}

// String renders a rule back to Datalog-ish text for explanation output.
func (r Rule) String() string {
	if len(r.Body) == 0 {
		return r.Head.String() + "."
	}
	parts := make([]string, len(r.Body))
	for i, a := range r.Body {
		parts[i] = a.String()
	}
	return r.Head.String() + " :- " + strings.Join(parts, ", ") + "."
}

// String renders an atom back to text.
func (a Atom) String() string {
	args := make([]string, len(a.Args))
	for i, t := range a.Args {
		args[i] = t.Name
	}
	prefix := ""
	if a.Neg {
		prefix = "!"
	}
	return prefix + a.Pred + "(" + strings.Join(args, ", ") + ")"
}
