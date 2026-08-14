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

// parseAtom parses a single (optionally negated) atom like `p(X, "c")`.
func parseAtom(s string) (Atom, error) {
	m := atomRE.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return Atom{}, fmt.Errorf("rules: malformed atom %q", s)
	}
	a := Atom{Pred: m[2], Neg: m[1] == "!"}
	for _, part := range splitArgs(m[3]) {
		a.Args = append(a.Args, parseTerm(part))
	}
	return a, nil
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
