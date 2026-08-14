package rules

import (
	"fmt"
	"regexp"
	"strings"
)

var atomRE = regexp.MustCompile(`^(!?)\s*([a-z_][a-zA-Z0-9_]*)\s*\((.*)\)$`)

// parseTerm classifies a token as a variable (uppercase-initial or "_") or a
// constant (quotes stripped).
func parseTerm(s string) Term {
	s = strings.TrimSpace(s)
	if s == "_" || (len(s) > 0 && s[0] >= 'A' && s[0] <= 'Z') {
		return Term{Name: s, Var: true}
	}
	return Term{Name: strings.Trim(s, `"'`), Var: false}
}

// Comparison predicates are represented as atoms whose Pred is one of these
// sentinels, carrying exactly two argument terms. They filter bindings rather
// than matching facts (see evalBody).
const (
	predEq  = "=cmp:eq"
	predNeq = "=cmp:neq"
)

// comparisonRE matches an infix comparison literal like `A != B` or `X = "c"`.
var comparisonRE = regexp.MustCompile(`^\s*(.+?)\s*(!=|=)\s*(.+?)\s*$`)

// parseAtom parses a single body element: either an infix comparison
// (`A != B`, `X = Y`) or an ordinary (optionally negated) atom like `p(X, "c")`.
func parseAtom(s string) (Atom, error) {
	s = strings.TrimSpace(s)
	// Try comparison first, but only when the string is not a predicate call
	// (a predicate call contains '(' before any operator).
	if !looksLikeCall(s) {
		if m := comparisonRE.FindStringSubmatch(s); m != nil {
			pred := predEq
			if m[2] == "!=" {
				pred = predNeq
			}
			return Atom{Pred: pred, Args: []Term{parseTerm(m[1]), parseTerm(m[3])}}, nil
		}
	}
	m := atomRE.FindStringSubmatch(s)
	if m == nil {
		return Atom{}, fmt.Errorf("rules: malformed atom %q", s)
	}
	a := Atom{Pred: m[2], Neg: m[1] == "!"}
	for _, part := range splitArgs(m[3]) {
		a.Args = append(a.Args, parseTerm(part))
	}
	return a, nil
}

// looksLikeCall reports whether s is a predicate call form `name(...)` rather
// than an infix comparison, by checking for a '(' that precedes any '=' / '!'.
func looksLikeCall(s string) bool {
	paren := strings.IndexByte(s, '(')
	if paren < 0 {
		return false
	}
	op := strings.IndexAny(s, "=!")
	return op < 0 || paren < op
}

// ParseRule parses a single clause: `head :- b1, b2.` or a bare fact `head.`.
// A trailing period is optional.
func ParseRule(line string) (Rule, error) {
	line = strings.TrimSpace(line)
	line = strings.TrimSuffix(line, ".")
	if line == "" {
		return Rule{}, fmt.Errorf("rules: empty clause")
	}
	var r Rule
	if strings.Contains(line, ":-") {
		parts := strings.SplitN(line, ":-", 2)
		h, err := parseAtom(parts[0])
		if err != nil {
			return r, err
		}
		r.Head = h
		// Aggregate rule form: head(...) :- count(V : source(...)) = K.
		if agg, ok, err := parseAggregate(parts[1], h); err != nil {
			return r, err
		} else if ok {
			r.Agg = agg
			return r, nil
		}
		for _, b := range splitTopLevel(parts[1]) {
			ba, err := parseAtom(b)
			if err != nil {
				return r, err
			}
			r.Body = append(r.Body, ba)
		}
		return r, nil
	}
	h, err := parseAtom(line)
	if err != nil {
		return r, err
	}
	r.Head = h
	return r, nil
}

// aggregateRE matches `count(V : source(...)) = K`.
var aggregateRE = regexp.MustCompile(`^\s*count\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.+)\)\s*=\s*([A-Za-z_][A-Za-z0-9_]*)\s*$`)

// parseAggregate recognises the aggregate body form
// `count(V : source(...)) = K` and builds the rule's aggregate descriptor.
// It returns ok=false when body is not an aggregate form.
func parseAggregate(body string, head Atom) (*aggregate, bool, error) {
	m := aggregateRE.FindStringSubmatch(strings.TrimSpace(body))
	if m == nil {
		return nil, false, nil
	}
	countVar, sourceStr, resultVar := m[1], m[2], m[3]
	source, err := parseAtom(sourceStr)
	if err != nil {
		return nil, false, fmt.Errorf("rules: aggregate source: %w", err)
	}
	// Find which head argument receives the count (the result variable K).
	resultOn := -1
	for i, t := range head.Args {
		if t.Var && t.Name == resultVar {
			resultOn = i
			break
		}
	}
	if resultOn < 0 {
		return nil, false, fmt.Errorf("rules: aggregate result variable %q does not appear in head %q", resultVar, head.Pred)
	}
	return &aggregate{countVar: countVar, source: source, resultOn: resultOn}, true, nil
}

// ParseProgram parses a multi-clause program, splitting on top-level periods
// and ignoring blank lines and `#`/`%` comment lines. It returns all parsed
// rules or the first error with its clause text for provenance.
func ParseProgram(src string) ([]Rule, error) {
	var rules []Rule
	for _, stmt := range splitClauses(src) {
		r, err := ParseRule(stmt)
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, nil
}

// splitClauses breaks a program into clause strings terminated by a top-level
// period, stripping comment lines first.
func splitClauses(src string) []string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "%") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	var out []string
	for _, stmt := range strings.Split(b.String(), ".") {
		if strings.TrimSpace(stmt) != "" {
			out = append(out, strings.TrimSpace(stmt))
		}
	}
	return out
}

// splitArgs splits a comma-separated argument list, respecting quotes.
func splitArgs(s string) []string {
	return splitRespecting(s, false)
}

// splitTopLevel splits body atoms on commas that are outside parens and quotes.
func splitTopLevel(s string) []string {
	return splitRespecting(s, true)
}

// splitRespecting splits s on commas, ignoring commas inside quotes and (when
// trackParens) inside parentheses.
func splitRespecting(s string, trackParens bool) []string {
	var out []string
	var cur strings.Builder
	depth := 0
	inq := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inq != 0:
			if c == inq {
				inq = 0
			}
			cur.WriteByte(c)
		case c == '"' || c == '\'':
			inq = c
			cur.WriteByte(c)
		case trackParens && c == '(':
			depth++
			cur.WriteByte(c)
		case trackParens && c == ')':
			depth--
			cur.WriteByte(c)
		case c == ',' && depth == 0:
			out = append(out, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	if strings.TrimSpace(cur.String()) != "" {
		out = append(out, cur.String())
	}
	return out
}
