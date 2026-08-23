package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestNNGuideDocumentsCanonicalHumanNavigationContract(t *testing.T) {
	guide, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-guide: %v", err)
	}
	content := string(guide)
	assertions := map[string][]string{
		"four action invariant": {
			"## Human-driven navigation invariant",
			"Recenter — move focus",
			"Peek — inspect without moving",
			"Scan — zoom out without moving",
			"Arrive — stop",
		},
		"canonical chooser": {
			"## Canonical navigation chooser",
			"offer at most four options",
			"During cold teleport, replace Recenter",
		},
		"presentation preflight": {
			"Before every chooser, present:",
			"Navigation presentation check:",
			"[ ] focus type and status shown",
			"[ ] Arrive available",
		},
		"arrive report": {
			"## Arrive report",
			"starting query or focus",
			"final note ID, type, and status",
			"region reached",
			"which paths remain unexplored",
			"focus remains at the final note",
			"understand the destination without opening the note separately",
			"2–3 sentences is a minimum for simple notes, not a cap",
			"Hub, model, protocol, or contested note",
			"A field-only checklist or one-line summary is not a compliant arrival",
		},
		"normative examples": {
			"Peek and Scan MUST be discoverable",
			"Bad:",
			"Why bad:",
			"Recenter ↓ checkpoint principle",
		},
		"navigation protocol": {
			"applies_when: human-driven nn graph navigation",
			"preserve focus for Peek and Scan",
			"change focus only after Teleport or Recenter",
		},
		"navigate resume shortcut": {
			"## `navigate` — resume navigation",
			"conversational shortcut, not an `nn` subcommand",
			"resume from the retained final focus",
			"re-run Orient",
			"If no focus is retained",
		},
	}
	for assertion, required := range assertions {
		for _, snippet := range required {
			if !strings.Contains(content, snippet) {
				t.Errorf("assertion %q failed: nn-guide missing %q", assertion, snippet)
			}
		}
	}
}
