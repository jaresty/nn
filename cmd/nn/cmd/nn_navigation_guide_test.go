package cmd

import (
	"os"
	"strings"
	"testing"

	nnSkills "github.com/jaresty/nn/skills"
	"gopkg.in/yaml.v3"
)

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	WhenToUse   string `yaml:"when_to_use"`
}

func parseSkillFrontmatter(t *testing.T, content string) skillFrontmatter {
	t.Helper()
	if !strings.HasPrefix(content, "---\n") {
		t.Fatal("skill is missing opening YAML frontmatter delimiter")
	}
	rest := strings.TrimPrefix(content, "---\n")
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		t.Fatal("skill is missing closing YAML frontmatter delimiter")
	}
	var frontmatter skillFrontmatter
	if err := yaml.Unmarshal([]byte(rest[:end]), &frontmatter); err != nil {
		t.Fatalf("parse skill frontmatter: %v", err)
	}
	return frontmatter
}

func TestNNNavigateSkillIsValidAndDiscoverable(t *testing.T) {
	data, err := os.ReadFile("../../../skills/nn-navigate/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-navigate: %v", err)
	}
	frontmatter := parseSkillFrontmatter(t, string(data))
	if frontmatter.Name != "nn-navigate" {
		t.Errorf("nn-navigate frontmatter name = %q, want nn-navigate", frontmatter.Name)
	}
	for _, required := range []string{"iterative", "human", "teleport", "orient", "recenter", "peek", "scan", "arrive", "history", "bookmarks"} {
		if !strings.Contains(strings.ToLower(frontmatter.Description), required) {
			t.Errorf("nn-navigate description missing activation term %q: %q", required, frontmatter.Description)
		}
	}

	embedded, err := nnSkills.FS.ReadFile("nn-navigate/SKILL.md")
	if err != nil {
		t.Fatalf("nn-navigate is not embedded: %v", err)
	}
	if string(embedded) != string(data) {
		t.Error("embedded nn-navigate differs from skills/nn-navigate/SKILL.md")
	}

	_, execute := setupNotebook(t)
	out, err := execute("skills", "list")
	if err != nil {
		t.Fatalf("nn skills list: %v", err)
	}
	if !strings.Contains(out, "nn-navigate") {
		t.Errorf("nn skills list does not discover nn-navigate: %q", out)
	}
	got, err := execute("skills", "get", "nn-navigate")
	if err != nil {
		t.Fatalf("nn skills get nn-navigate: %v", err)
	}
	if got != string(data) {
		t.Error("nn skills get nn-navigate did not return the shipped skill")
	}
}

func TestNNNavigateOwnsCanonicalHumanNavigationContract(t *testing.T) {
	data, err := os.ReadFile("../../../skills/nn-navigate/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-navigate: %v", err)
	}
	content := string(data)
	assertions := map[string][]string{
		"preflight and navigation mode": {
			"## Preflight and activation",
			"nn skills list",
			"nn skills get nn-navigate",
			"### Navigation mode: the zoned navigator with contents",
			"Enter", "Orient", "Read from here", "Recenter", "Arrive",
		},
		"zones as navigation model": {
			"TOP (what it answers to)",
			"BOTTOM (what builds on it)",
			"LEFT (tension)",
			"RIGHT (provenance)",
			"Empty zones are information too",
		},
		"four action invariant": {
			"## Human-driven navigation invariant",
			"Recenter — move focus",
			"Peek — inspect without moving",
			"Scan — zoom out without moving",
			"Arrive — stop",
		},
		"canonical chooser and presentation": {
			"## Canonical navigation chooser",
			"offer at most four options",
			"During cold teleport, replace Recenter",
			"Peek and Scan MUST be discoverable",
			"Before every chooser, present:",
			"Navigation presentation check:",
			"[ ] focus type and status shown",
			"[ ] zone/type/edge color markers applied (stable relay palette)",
			"[ ] zone positions carry their color markers in the map",
			"[ ] legend/key for the color markers shown so they are interpretable",
			"[ ] Arrive available",
			"Bad:",
			"Why bad:",
			"Recenter ↓ checkpoint principle",
			"#### Presentation discipline (the named block every seam cites)",
			"P1 — Colors and relay budgets on",
			"P2 — Focus + Map + Moves",
			"P3 — Canonical four-action chooser",
			"P4 — Degree-scaled summaries",
			"--zones --bodies --presentation-hints --color always",
		},
		"compact enforcement seed": {
			"applies_when: human-driven nn graph navigation",
			"preserve focus for Peek and Scan",
			"adopt a new destination only after a successful Teleport, Visit, Recenter, or Go to",
			"relationship family is known but the destination is unknown",
			"carry the complete navigation state through compaction",
		},
		"arrival scaling": {
			"## Arrive report",
			"starting query or focus",
			"final note ID, type, and status",
			"region reached",
			"which paths remain unexplored",
			"focus remains at the final note",
			"### Arrival depth",
			"understand the destination without opening the note separately",
			"2–3 sentences is a minimum for simple notes, not a cap",
			"Leaf or simple note",
			"Connected note",
			"Hub, model, protocol, or contested note",
			"A field-only checklist or one-line summary is not a compliant arrival",
		},
		"route impact and path overlays": {
			"#### Typed destination discovery",
			"nn graph routes --focus ID --links TYPES --search QUERY --limit N --json",
			"#### Explicit typed impact overlay",
			"nn graph impact --focus ID --links TYPES --direction incoming|outgoing --depth N --json",
			"#### Typed path route overlay",
			"nn path <a> <b> --links <types> --json",
			"nodes[1]",
		},
		"teleport scan and peek": {
			"#### `scan` — look wide",
			"Your territory (ego, depth 2)",
			"The wider landscape (global)",
			"nn graph bridges --search \"<query>\" --format json",
			"--exclude <focus-id>",
			"bounded crossing witnesses",
			"not proof of territorial separation",
			"#### `peek` — look deep without moving",
			"focus stays put",
			"#### `teleport` — move far",
			"default landing-zone source",
			"selected landing automatically becomes the retained focus",
			"Immediately run Orient",
			"present Focus + Map + Moves",
			"MUST NOT ask for a second confirmation",
			"MUST NOT offer a separate `Visit` action",
			"genuinely ambiguous",
		},
		"resume history bookmarks and compaction": {
			"## `navigate` — resume navigation",
			"conversational shortcut, not an `nn` subcommand",
			"resume from the retained final focus",
			"re-run Orient",
			"If no focus is retained",
			"## Conversational navigation history and bookmarks",
			"retained focus plus its active traversal context and filters",
			"conversation-scoped state, not `nn` subcommands",
			"successful Teleport, Visit, Recenter, or Go to",
			"push the prior frame onto Back and clear Forward",
			"initial landing cannot push a frame",
			"moves the current frame onto Forward and restores the latest Back frame",
			"moves the current frame onto Back and restores the latest Forward frame",
			"Bookmark <name>",
			"case-sensitive name",
			"explicit confirmation before replacing",
			"Go to <name>",
			"restores its saved frame as a Teleport landing",
			"Where am I?",
			"immediate Back and Forward destinations",
			"does not mutate navigation state",
			"Failed or no-op operations never mutate",
			"Back stack is empty",
			"unknown bookmark",
			"full current frame, Back and Forward stacks, and every bookmark",
			"state is unknown: never invent",
		},
	}
	for assertion, required := range assertions {
		for _, snippet := range required {
			if !strings.Contains(content, snippet) {
				t.Errorf("assertion %q failed: nn-navigate missing %q", assertion, snippet)
			}
		}
	}
}

func TestNNGuideDispatchesHumanNavigationWithoutDuplicatingOwner(t *testing.T) {
	data, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-guide: %v", err)
	}
	content := string(data)
	frontmatter := parseSkillFrontmatter(t, content)
	for _, required := range []string{"command", "flag", "reference", "nn-navigate"} {
		if !strings.Contains(strings.ToLower(frontmatter.Description), required) {
			t.Errorf("nn-guide description missing routing term %q: %q", required, frontmatter.Description)
		}
	}
	for _, required := range []string{
		"### Human-driven iterative navigation: dispatch to `nn-navigate`",
		"iterative", "human-driven", "positioned focus",
		"nn skills list", "nn skills get nn-navigate",
		"teleport", "orient", "recenter", "peek", "scan", "arrive",
		"history", "bookmarks", "compaction",
	} {
		if !strings.Contains(strings.ToLower(content), strings.ToLower(required)) {
			t.Errorf("nn-guide navigation dispatch missing %q", required)
		}
	}

	for _, ownerOnly := range []string{
		"## Human-driven navigation invariant",
		"## Canonical navigation chooser",
		"## Arrive report",
		"## Conversational navigation history and bookmarks",
		"#### `scan` — look wide",
		"#### Typed destination discovery",
		"#### Explicit typed impact overlay",
		"#### Typed path route overlay",
		"#### `peek` — look deep without moving",
		"#### `teleport` — move far",
		"#### Presentation discipline (the named block every seam cites)",
		"selected landing automatically becomes the retained focus",
		"2–3 sentences is a minimum for simple notes, not a cap",
		"push the prior frame onto Back and clear Forward",
	} {
		if strings.Contains(content, ownerOnly) {
			t.Errorf("nn-guide duplicates nn-navigate owner contract %q", ownerOnly)
		}
	}
}

func TestNNGuidePreservesGraphCommandReference(t *testing.T) {
	data, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-guide: %v", err)
	}
	content := string(data)
	for _, required := range []string{
		"nn graph show [--focus <id>]",
		"nn graph routes --focus ID --links TYPES --search QUERY --limit N --json",
		"nn graph impact --focus ID --links TYPES --direction incoming|outgoing --depth N --json",
		"nn path <a> <b> --links <types>",
		"nn graph bridges [--search \"<query>\"]",
		"nn clusters --search \"<query>\" --json",
		"grounded-by OUT → TOP",
		"supports OUT → BOTTOM",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("nn-guide command reference missing %q", required)
		}
	}
}

func TestNNNavigateOwnsStableColorRelayDiscipline(t *testing.T) {
	ownerData, err := os.ReadFile("../../../skills/nn-navigate/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-navigate: %v", err)
	}
	guideData, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-guide: %v", err)
	}
	owner, guide := string(ownerData), string(guideData)
	for _, required := range []string{
		"Stable emoji relay palette",
		"every color-capable human-facing navigation view",
		"`🔵 TOP`", "`🟢 BOTTOM`", "`🔴 LEFT`", "`🔷 RIGHT`", "`🟠 FOCUS / REGION`",
		"pre-landing Teleport chooser",
		"Scan",
		"Graph text sources MUST use `--color always`",
		"JSON sources are marker-free",
		"manually apply the relay palette",
		"reproduce this legend",
		"Plain, uncolored Focus + Map + Moves is noncompliant",
	} {
		if !strings.Contains(owner, required) {
			t.Errorf("nn-navigate color relay discipline missing %q", required)
		}
	}
	for _, ownerOnly := range []string{
		"Stable emoji relay palette",
		"`🔵 TOP`", "`🟢 BOTTOM`", "`🔴 LEFT`", "`🔷 RIGHT`", "`🟠 FOCUS / REGION`",
		"Plain, uncolored Focus + Map + Moves is noncompliant",
	} {
		if strings.Contains(guide, ownerOnly) {
			t.Errorf("nn-guide duplicates nn-navigate color relay detail %q", ownerOnly)
		}
	}

	_, execute := setupNotebook(t)
	virtual, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("show virtual-nn-cli-reference: %v", err)
	}
	if !strings.Contains(virtual, "nn-navigate enforces the color relay discipline") {
		t.Error("virtual CLI reference does not dispatch color relay discipline to nn-navigate")
	}
	for _, ownerOnly := range []string{"`🔵 TOP`", "`🟢 BOTTOM`", "`🔴 LEFT`", "`🔷 RIGHT`", "`🟠 FOCUS / REGION`"} {
		if strings.Contains(virtual, ownerOnly) {
			t.Errorf("virtual CLI reference duplicates nn-navigate palette %q", ownerOnly)
		}
	}
}

func TestVirtualCLIReferenceDispatchesNavigationWithoutOwningWorkflow(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("show virtual-nn-cli-reference: %v", err)
	}
	for _, required := range []string{
		"Before human-driven iterative navigation",
		"if you have not yet done so this session, run `nn skills list`",
		"run `nn skills get nn-navigate`",
		"binding skill dispatch",
		"nn graph show",
		"nn graph routes",
		"nn graph impact",
		"nn path",
	} {
		if !strings.Contains(out, required) {
			t.Errorf("virtual CLI reference navigation dispatch missing %q", required)
		}
	}
	for _, ownerOnly := range []string{
		"Each successful Teleport, Visit, Recenter, or Go to",
		"Bookmark names are case-sensitive",
		"2–3 sentences is a minimum for simple notes, not a cap",
		"Presentation discipline (the named block every seam cites)",
	} {
		if strings.Contains(out, ownerOnly) {
			t.Errorf("virtual CLI reference duplicates nn-navigate contract %q", ownerOnly)
		}
	}
}
