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
			"--presentation-hints",
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
			"adopt a new destination only after a successful Teleport, Visit, Recenter, or Go to",
			"--zones --bodies --presentation-hints --color always",
			"relationship family is known but the destination is unknown",
			"nn graph routes --focus ID --links TYPES --search QUERY --limit N --json",
		},
		"navigate resume shortcut": {
			"## `navigate` — resume navigation",
			"conversational shortcut, not an `nn` subcommand",
			"resume from the retained final focus",
			"re-run Orient",
			"If no focus is retained",
		},
		"teleport automatic landing": {
			"selected landing automatically becomes the retained focus",
			"Immediately run Orient",
			"present Focus + Map + Moves",
			"MUST NOT ask for a second confirmation",
			"MUST NOT offer a separate `Visit` action",
			"genuinely ambiguous",
		},
		"conversational navigation frame": {
			"## Conversational navigation history and bookmarks",
			"retained focus plus its active traversal context and filters",
			"conversation-scoped state, not `nn` subcommands",
		},
		"history transitions": {
			"successful Teleport, Visit, Recenter, or Go to",
			"push the prior frame onto Back and clear Forward",
			"initial landing cannot push a frame",
			"moves the current frame onto Forward and restores the latest Back frame",
			"moves the current frame onto Back and restores the latest Forward frame",
			"rerun Orient and present Focus + Map + Moves",
		},
		"bookmarks and inspection": {
			"Bookmark <name>",
			"case-sensitive name",
			"explicit confirmation before replacing",
			"Go to <name>",
			"restores its saved frame as a Teleport landing",
			"Where am I?",
			"immediate Back and Forward destinations",
			"does not mutate navigation state",
		},
		"failure and continuity": {
			"Failed or no-op operations never mutate",
			"Back stack is empty",
			"unknown bookmark",
			"full current frame, Back and Forward stacks, and every bookmark",
			"state is unknown: never invent",
		},
	}
	if !strings.Contains(content, "[--presentation-hints]") {
		t.Errorf("graph-show syntax missing --presentation-hints")
	}
	embedded, err := os.ReadFile("show.go")
	if err != nil {
		t.Fatalf("read embedded CLI reference: %v", err)
	}
	embeddedContent := string(embedded)
	for _, required := range []string{
		"match_density",
		"match_count / size",
		"explanatory signal, not a ranking input",
		"Navigation frame = retained focus plus active traversal context and filters",
		"pushes the prior frame onto Back and clears Forward",
		"Back moves current to Forward",
		"Forward moves current to Back",
		"Bookmark names are case-sensitive",
		"explicit confirmation",
		"Where am I? reports",
		"Failed and no-op operations never mutate history or bookmarks",
		"compaction handoff",
		"state is unknown; never invent it",
	} {
		if !strings.Contains(embeddedContent, required) {
			t.Errorf("embedded CLI reference missing %q", required)
		}
	}
	for assertion, required := range assertions {
		for _, snippet := range required {
			if !strings.Contains(content, snippet) {
				t.Errorf("assertion %q failed: nn-guide missing %q", assertion, snippet)
			}
		}
	}
}
