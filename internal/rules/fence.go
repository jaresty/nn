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
	var buf []string
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case !inFence && strings.HasPrefix(trimmed, "```nn-rule"):
			inFence = true
			buf = buf[:0]
		case inFence && strings.HasPrefix(trimmed, "```"):
			inFence = false
			for _, stmt := range splitClauses(strings.Join(buf, "\n")) {
				r, err := ParseRule(stmt)
				if err != nil {
					warns = append(warns, fmt.Sprintf("note %s: skipped malformed rule %q: %v", noteID, stmt, err))
					continue
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
