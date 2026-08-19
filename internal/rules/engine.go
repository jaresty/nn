// Package rules implements a small pure-Go semi-naive Datalog engine that
// evaluates rules over facts derived from notes. It is a pure derivation layer:
// facts and rules both come from the Markdown on disk, and the engine never
// mutates truth — it only computes derived facts (per ADR-0019).
package rules

import (
	"fmt"
	"maps"
	"math"
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
	kind     string // count | min | max | sum
	countVar string // the variable aggregated (V)
	source   Atom   // the single source literal
	resultOn int    // index into Head.Args that receives the result (K)
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
	// byPred indexes facts by predicate so evalBody and Query iterate only the
	// facts that can match a given literal, rather than scanning the whole base.
	byPred map[string][]Fact
	rules  []Rule
	// deriv records, for each derived fact key, one witness of how it was
	// produced: the rule that fired and the premise fact keys it consumed. Base
	// facts (added via AddFact) are absent from this map.
	deriv map[string]derivation
	// evalCount counts completed Eval() calls, so callers can verify the
	// fixpoint runs once per command rather than once per queried note.
	evalCount int
}

// EvalCount reports how many times Eval has completed on this engine.
func (e *Engine) EvalCount() int { return e.evalCount }

// derivation is one witness of how a derived fact was produced.
type derivation struct {
	rule     Rule
	premises []string // fact keys
}

// NewEngine returns an empty engine.
func NewEngine() *Engine {
	return &Engine{facts: map[string]Fact{}, byPred: map[string][]Fact{}, deriv: map[string]derivation{}}
}

// addFact inserts a ground fact into both the keyed base and the predicate
// index. It is the single insertion point that keeps byPred consistent with
// facts; callers must not write e.facts directly.
func (e *Engine) addFact(f Fact) {
	k := f.Key()
	if _, ok := e.facts[k]; ok {
		return
	}
	e.facts[k] = f
	e.byPred[f.Pred] = append(e.byPred[f.Pred], f)
}

// AddFact inserts a ground fact into the fact base.
func (e *Engine) AddFact(f Fact) { e.addFact(f) }

// AddRules appends rules to the ruleset.
func (e *Engine) AddRules(rs []Rule) { e.rules = append(e.rules, rs...) }

// has reports whether a ground fact is present.
func (e *Engine) has(f Fact) bool { _, ok := e.facts[f.Key()]; return ok }

// Query returns all facts for the given predicate, sorted by key.
func (e *Engine) Query(pred string) []Fact {
	out := append([]Fact(nil), e.byPred[pred]...)
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out
}

// Eval validates and evaluates the complete ruleset.
func (e *Engine) Eval() error { return e.evalRules(e.rules) }

// EvalFor evaluates only rules in the transitive dependency closure of the
// requested predicates. Dependencies include positive, negated, recursive, and
// aggregate source predicates. Base predicates need no deriving rule.
func (e *Engine) EvalFor(predicates ...string) error {
	return e.evalRules(dependencyRules(e.rules, predicates))
}

func dependencyRules(all []Rule, predicates []string) []Rule {
	byHead := make(map[string][]int)
	for i, r := range all {
		byHead[r.Head.Pred] = append(byHead[r.Head.Pred], i)
	}

	neededPred := make(map[string]bool)
	neededRule := make(map[int]bool)
	queue := append([]string(nil), predicates...)
	for len(queue) > 0 {
		pred := queue[0]
		queue = queue[1:]
		if neededPred[pred] {
			continue
		}
		neededPred[pred] = true
		for _, idx := range byHead[pred] {
			if neededRule[idx] {
				continue
			}
			neededRule[idx] = true
			r := all[idx]
			if r.Agg != nil {
				queue = append(queue, r.Agg.source.Pred)
				continue
			}
			for _, lit := range r.Body {
				if !isComparison(lit.Pred) {
					queue = append(queue, lit.Pred)
				}
			}
		}
	}

	selected := make([]Rule, 0, len(neededRule))
	for i, r := range all {
		if neededRule[i] {
			selected = append(selected, r)
		}
	}
	return selected
}

// evalRules validates safety and stratification, then computes the least
// fixpoint for exactly the supplied rules.
func (e *Engine) evalRules(selected []Rule) error {
	if err := checkSafe(selected); err != nil {
		return err
	}
	if err := checkStratified(selected); err != nil {
		return err
	}
	// Interleave ordinary fixpoint and aggregate evaluation until neither adds a
	// fact. Aggregates read the completed source relation each round; downstream
	// rules then see the aggregate's output on the next fixpoint pass.
	for {
		e.fixpoint(selected)
		if !e.evalAggregates(selected) {
			e.evalCount++
			return nil
		}
	}
}

// evalAggregates runs selected aggregate rules against the current facts. It
// returns whether any new fact was added.
func (e *Engine) evalAggregates(selected []Rule) bool {
	added := false
	for _, r := range selected {
		if r.Agg == nil {
			continue
		}
		for _, f := range e.aggregateFacts(r) {
			if !e.has(f) {
				e.addFact(f)
				added = true
			}
		}
	}
	return added
}

// aggregateFacts computes the head facts produced by an aggregate rule: for each
// group (distinct binding of the head's non-result variables), fold the values
// of the aggregated variable among matching source facts. count uses the number
// of DISTINCT values; sum/min/max fold the numeric values (each source fact
// contributes once, so a multiset — duplicates matter for sum). For sum/min/max
// non-numeric values are skipped, and a group with no numeric values yields no
// head fact.
func (e *Engine) aggregateFacts(r Rule) []Fact {
	agg := r.Agg
	// group key -> ordered list of the aggregated variable's values (multiset).
	groupVals := map[string][]string{}
	groupArgs := map[string][]string{}
	var order []string // stable group iteration order

	for _, f := range e.facts {
		b, ok := unify(agg.source, f, bindings{})
		if !ok {
			continue
		}
		val, bound := b[agg.countVar]
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
		if _, seen := groupArgs[key]; !seen {
			groupArgs[key] = args
			order = append(order, key)
		}
		groupVals[key] = append(groupVals[key], val)
	}

	var out []Fact
	for _, key := range order {
		res, ok := foldAggregate(agg.kind, groupVals[key])
		if !ok {
			continue // e.g. sum/min/max over a group with no numeric values
		}
		args := append([]string(nil), groupArgs[key]...)
		args[agg.resultOn] = res
		out = append(out, Fact{Pred: r.Head.Pred, Args: args})
	}
	return out
}

// foldAggregate reduces a group's aggregated values by kind, returning the
// formatted result and whether a result exists. count = number of distinct
// values (always ok). sum/min/max operate on the numeric values; non-numeric
// values are skipped, and ok=false if no numeric value remains.
func foldAggregate(kind string, vals []string) (string, bool) {
	if kind == "count" {
		distinct := map[string]bool{}
		for _, v := range vals {
			distinct[v] = true
		}
		return strconv.Itoa(len(distinct)), true
	}
	var acc float64
	have := false
	for _, v := range vals {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			continue
		}
		switch {
		case !have:
			acc = f
		case kind == "min" && f < acc:
			acc = f
		case kind == "max" && f > acc:
			acc = f
		case kind == "sum":
			acc += f
		}
		have = true
	}
	if !have {
		return "", false
	}
	return formatNumber(acc), true
}

// formatNumber renders an aggregate result, using an integer form when the value
// is integral so `sum = 5` prints "5" rather than "5.000000".
func formatNumber(f float64) string {
	if f == math.Trunc(f) && !math.IsInf(f, 0) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// checkSafe rejects rules where a comparison literal references a variable that
// is not bound by any positive (non-comparison, non-negated) body literal —
// such a comparison is not range-restricted and cannot be evaluated.
func checkSafe(rules []Rule) error {
	for _, r := range rules {
		bound := map[string]bool{}
		for _, lit := range r.Body {
			if lit.Neg || isComparison(lit.Pred) {
				continue
			}
			for _, t := range lit.Args {
				if t.Var && t.Name != "_" {
					bound[t.Name] = true
				}
			}
		}
		for _, lit := range r.Body {
			if !isComparison(lit.Pred) {
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

// positivePredIdxs returns the body-literal indices of a rule's positive
// (non-negated, non-comparison) literals — the join positions that a semi-naive
// delta round can restrict to newly-derived facts. A rule with no such literal,
// or with any negated literal, is not eligible for delta evaluation (see
// fixpoint): negation is only correct against a saturated relation, so such
// rules are re-evaluated fully every round.
func positivePredIdxs(body []Atom) (idxs []int, hasNeg bool) {
	for i, lit := range body {
		switch {
		case isComparison(lit.Pred):
			// comparison: not a join position
		case lit.Neg:
			hasNeg = true
		default:
			idxs = append(idxs, i)
		}
	}
	return idxs, hasNeg
}

// fixpoint iterates the rules until no new fact is derived, using semi-naive
// (delta) evaluation for negation-free rules.
//
// Naive evaluation re-joins every rule over the whole fact base each round,
// re-deriving all previously-known facts. Semi-naive instead tracks the facts
// added in the previous round (the delta) and, for a negation-free rule,
// requires at least one positive body literal to bind against a delta fact — so
// a rule fires only when it can produce something new. Rules containing a
// negated literal are evaluated fully every round: their correctness depends on
// the negated relation being saturated, which the outer convergence loop
// guarantees but a delta round does not. Both paths compute the same least
// fixpoint; only redundant re-derivation is eliminated.
func (e *Engine) fixpoint(selected []Rule) {
	// Round 0: evaluate every rule fully. Seed the next delta with everything
	// added here (plus the base facts, so first-round consumers of base facts
	// via delta rounds still see them).
	delta := map[string][]Fact{}
	for pred, fs := range e.byPred {
		delta[pred] = append([]Fact(nil), fs...)
	}
	e.applyRules(selected, func(r Rule) []bindings { return e.evalBody(r.Body) }, &delta)

	for len(delta) > 0 {
		next := map[string][]Fact{}
		for _, r := range selected {
			if r.Agg != nil {
				continue
			}
			idxs, hasNeg := positivePredIdxs(r.Body)
			var results []bindings
			if hasNeg || len(idxs) == 0 {
				// Not delta-eligible: full evaluation (cheap — few such facts).
				results = e.evalBody(r.Body)
			} else {
				// Semi-naive: the union over each positive literal position of
				// the body evaluated with that position restricted to delta.
				// De-dup identical head facts across positions via addFact.
				seen := map[string]bool{}
				for _, di := range idxs {
					if len(delta[r.Body[di].Pred]) == 0 {
						continue
					}
					for _, b := range e.evalBodyDelta(r.Body, di, delta) {
						key := subst(r.Head, b).Key()
						if seen[key] {
							continue
						}
						seen[key] = true
						results = append(results, b)
					}
				}
			}
			e.deriveInto(r, results, &next)
		}
		delta = next
	}
}

// applyRules runs eval(r) for every non-aggregate rule and records new head
// facts, seeding *nextDelta with the facts added.
func (e *Engine) applyRules(rules []Rule, eval func(Rule) []bindings, nextDelta *map[string][]Fact) {
	for _, r := range rules {
		if r.Agg != nil {
			continue
		}
		e.deriveInto(r, eval(r), nextDelta)
	}
}

// deriveInto grounds each binding into a head fact; newly-added facts are
// recorded (witness in e.deriv) and pushed onto *nextDelta for the next round.
func (e *Engine) deriveInto(r Rule, bs []bindings, nextDelta *map[string][]Fact) {
	for _, b := range bs {
		nf := subst(r.Head, b)
		if !e.has(nf) {
			e.addFact(nf)
			e.deriv[nf.Key()] = derivation{rule: r, premises: positivePremiseKeys(r.Body, b)}
			(*nextDelta)[nf.Pred] = append((*nextDelta)[nf.Pred], nf)
		}
	}
}

// evalBodyDelta is evalBody with the positive literal at deltaIdx iterating only
// the delta facts for its predicate, instead of the full fact base. All other
// positive literals still range over the full fact base. This is the semi-naive
// rule instantiation: the head can be new only if the deltaIdx literal matched a
// fact derived in the previous round.
func (e *Engine) evalBodyDelta(body []Atom, deltaIdx int, delta map[string][]Fact) []bindings {
	results := []bindings{{}}
	for i, lit := range body {
		var next []bindings
		for _, b := range results {
			switch {
			case isComparison(lit.Pred):
				if compareHolds(lit, b) {
					next = append(next, b)
				}
			case lit.Neg:
				if !e.has(subst(lit, b)) {
					next = append(next, b)
				}
			default:
				facts := e.byPred[lit.Pred]
				if i == deltaIdx {
					facts = delta[lit.Pred]
				}
				for _, f := range facts {
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
			case isComparison(lit.Pred):
				if compareHolds(lit, b) {
					next = append(next, b)
				}
			case lit.Neg:
				if !e.has(subst(lit, b)) {
					next = append(next, b)
				}
			default:
				for _, f := range e.byPred[lit.Pred] {
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

// compareHolds evaluates a comparison literal under bindings b. Both operands
// must be bound to a constant (variable resolved via b, or a literal constant);
// an unbound operand is treated as a failed comparison.
//
// Equality (=) and inequality (!=) compare operands as strings. The ordering
// operators (<, <=, >, >=) compare them numerically: both operands must parse as
// numbers, and a non-numeric operand makes the comparison fail (never errors),
// matching the unbound-operand convention.
func compareHolds(lit Atom, b bindings) bool {
	l, lok := resolveTerm(lit.Args[0], b)
	r, rok := resolveTerm(lit.Args[1], b)
	if !lok || !rok {
		return false
	}
	switch lit.Pred {
	case predEq:
		return l == r
	case predNeq:
		return l != r
	}
	lf, lerr := strconv.ParseFloat(l, 64)
	rf, rerr := strconv.ParseFloat(r, 64)
	if lerr != nil || rerr != nil {
		return false
	}
	switch lit.Pred {
	case predLt:
		return lf < rf
	case predLte:
		return lf <= rf
	case predGt:
		return lf > rf
	case predGte:
		return lf >= rf
	}
	return false
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
	// Check every argument against the incoming bindings (and against earlier
	// args of this same atom, tracked in pending) before allocating nb. This
	// avoids copying b for facts that fail to unify — the common case when a
	// literal shares a predicate with many non-matching facts.
	var pending []struct{ k, v string }
	for i, t := range a.Args {
		switch {
		case t.Var && t.Name == "_":
			// wildcard: matches anything, binds nothing
		case t.Var:
			cur, ok := b[t.Name]
			if !ok {
				for _, p := range pending {
					if p.k == t.Name {
						cur, ok = p.v, true
						break
					}
				}
			}
			if ok {
				if cur != f.Args[i] {
					return nil, false
				}
			} else {
				pending = append(pending, struct{ k, v string }{t.Name, f.Args[i]})
			}
		default:
			if t.Name != f.Args[i] {
				return nil, false
			}
		}
	}
	nb := make(bindings, len(b)+len(pending))
	maps.Copy(nb, b)
	for _, p := range pending {
		nb[p.k] = p.v
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
			if isComparison(lit.Pred) {
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
