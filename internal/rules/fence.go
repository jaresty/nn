package rules

import (
	"fmt"
	"strings"
)

// ExtractFenceRules scans a note body for ```nn-rule fenced code blocks and
// parses each clause inside them. A malformed clause never aborts loading: the
// well-formed rules are still returned, and a human-readable warning naming the
// note ID (for provenance) is appended to warns. This upholds the ADR-0019
// requirement that a bad rule must never break note loading.
func ExtractFenceRules(noteID, body string) (rules []Rule, warns []string) {
	inFence := false
	var scope string // representation scope from the current fence header, if any
	var buf []string
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case !inFence && strings.HasPrefix(trimmed, "```nn-rule"):
			inFence = true
			scope = parseFenceScope(trimmed)
			buf = buf[:0]
		case inFence && strings.HasPrefix(trimmed, "```"):
			inFence = false
			for _, stmt := range splitClauses(strings.Join(buf, "\n")) {
				r, err := ParseRule(stmt)
				if err != nil {
					warns = append(warns, fmt.Sprintf("note %s: skipped malformed rule %q: %v", noteID, stmt, err))
					continue
				}
				if scope != "" {
					if w := applyScope(&r, scope); w != "" {
						warns = append(warns, fmt.Sprintf("note %s: %s", noteID, w))
					}
				}
				rules = append(rules, r)
			}
		case inFence:
			buf = append(buf, line)
		}
	}
	if inFence {
		warns = append(warns, fmt.Sprintf("note %s: unclosed ```nn-rule fence", noteID))
	}
	return rules, warns
}

// parseFenceScope extracts the representation from a fence header of the form
// "```nn-rule scope=<rep>", returning "" when no scope is present.
func parseFenceScope(header string) string {
	rest := strings.TrimSpace(strings.TrimPrefix(header, "```nn-rule"))
	for _, tok := range strings.Fields(rest) {
		if v, ok := strings.CutPrefix(tok, "scope="); ok {
			return v
		}
	}
	return ""
}

// applyScope injects a representation(V, rep) guard into r's body, where V is the
// first variable argument of the head — restricting the rule to notes with that
// representation. It returns a non-empty warning (and leaves r unscoped) when the
// rule cannot be scoped: an aggregate rule, or a head with no variable argument.
func applyScope(r *Rule, rep string) string {
	if r.Agg != nil {
		return fmt.Sprintf("scope=%s ignored on aggregate rule %q (loaded unscoped)", rep, r.Head.Pred)
	}
	subject := ""
	for _, t := range r.Head.Args {
		if t.Var && t.Name != "_" {
			subject = t.Name
			break
		}
	}
	if subject == "" {
		return fmt.Sprintf("scope=%s ignored on rule %q with no variable head argument (loaded unscoped)", rep, r.Head.Pred)
	}
	r.Body = append(r.Body, Atom{
		Pred: "representation",
		Args: []Term{{Name: subject, Var: true}, {Name: rep, Var: false}},
	})
	return ""
}
