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

func TestNNNavigateOwnsAdaptiveHumanNavigationContract(t *testing.T) {
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
			"🔵 TOP — what the focus answers to",
			"🟢 BOTTOM — what builds on the focus",
			"🔴 LEFT — what contests or questions the focus",
			"🔷 RIGHT — lateral provenance or task relationships",
			"Empty zones are information too",
		},
		"four action invariant": {
			"## Human-driven navigation invariant",
			"Recenter — move focus",
			"Peek — inspect without moving",
			"Scan — zoom out without moving",
			"Arrive — stop",
		},
		"adaptive picker and presentation": {
			"## Adaptive hierarchical quick-actions picker",
			"top-level picker has at most four rows",
			"up to two evidence-backed contextual concrete shortcuts",
			"stable **`Lenses…`** row",
			"final row is always `All navigation actions…`",
			"Peek and Scan MUST be discoverable",
			"Before every picker, present:",
			"Navigation presentation check:",
			"[ ] focus type and status shown",
			"[ ] zone/type/edge color markers applied (stable relay palette)",
			"[ ] zone positions carry their color markers in the map",
			"[ ] legend/key for the color markers shown so they are interpretable",
			"[ ] Arrive available one level away",
			"Bad:",
			"Why bad:",
			"Recenter 🔵 TOP — what this focus answers to: move to <id> — <readable target title> because <substantive body-derived reason>",
			"#### Presentation discipline (the named block every seam cites)",
			"P1 — Colors and relay budgets on",
			"P2 — Focus + Map + Moves",
			"P3 — Adaptive hierarchical quick-actions picker",
			"P4 — Degree-scaled summaries",
			"--zones --bodies --presentation-hints --color always",
		},
		"compact enforcement seed": {
			"applies_when: human-driven nn graph navigation",
			"preserve focus for Peek and Scan",
			"adopt a new destination only after a successful Teleport, Visit, Recenter, or Go to",
			"relationship family is known but the destination is unknown",
			"carry the complete navigation and menu UI state through compaction",
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

func TestNNNavigateOwnsAdaptiveHierarchicalQuickActionsPicker(t *testing.T) {
	data, err := os.ReadFile("../../../skills/nn-navigate/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-navigate: %v", err)
	}
	content := string(data)

	pickerStart := strings.Index(content, "## Adaptive hierarchical quick-actions picker")
	pickerEnd := strings.Index(content, "## Arrive report")
	if pickerStart < 0 || pickerEnd <= pickerStart {
		t.Fatal("nn-navigate adaptive hierarchical quick-actions picker section is missing or malformed")
	}
	picker := content[pickerStart:pickerEnd]
	for _, required := range []string{
		"top-level picker has at most four rows",
		"up to two evidence-backed contextual concrete shortcuts",
		"stable **`Lenses…`** row",
		"final row is always `All navigation actions…`",
		"names its action class (`Recenter`, `Peek`, `Scan`, or `Lens`)",
		"states whether selecting it changes or retains focus",
		"body- or evidence-derived reason",
		"Generic availability is not evidence",
		"executes directly rather than opening its category submenu",
		"A specific promoted `Lens` action may occupy a contextual shortcut slot",
		"the stable `Lenses…` row remains present",
		"Every picker and submenu has at most four rows.",
		"Esc or a declined chooser in a submenu returns to the parent menu",
		"without mutating focus, graph history, notes, links, the goal, filters, traversal context, or notebook content",
		"Never render an explicit `Back` row.",
	} {
		if !strings.Contains(picker, required) {
			t.Errorf("adaptive picker contract missing %q", required)
		}
	}

	exactMenus := []string{
		"1. `Recenter`\n2. `Peek`\n3. `Scan`\n4. `■ Arrive`",
		"1. `○ Show verbatim`\n2. `○ Explain in depth`",
		"1. `○ Analogize`\n2. `↗ Find an analog`\n3. `○ Visualize`\n4. `○ Quiz`",
		"1. `○ Local territory`\n2. `↗ Global landscape`",
	}
	for _, menu := range exactMenus {
		if !strings.Contains(picker, menu) {
			t.Errorf("adaptive picker missing exact submenu:\n%s", menu)
		}
	}
	for _, required := range []string{
		"`All navigation actions…` opens the action-class submenu exactly one level away",
		"The `Lenses…` row is always present at the top level and opens this exact submenu one picker level away",
		"Show verbatim and Explain in depth remain Peek actions",
		"Find an analog is a human-intent Lens",
		"internally uses Scan retrieval",
		"Scan contains exactly Local territory and Global landscape",
		"`Recenter` may open `<short-id> · Quick actions › All actions › Recenter destinations`",
		"Before every picker, present:",
		"A. **Focus**", "B. **Map**", "C. **Moves**",
		"This **MUST PRESENT** rule",
		"→ focus changes; ○ focus retained; ■ stops; ↗ explores beyond local",
		"Every concrete action row includes exactly one applicable effect marker",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("adaptive positioned-view contract missing %q", required)
		}
	}
	for _, contradictory := range []string{
		"## Canonical navigation chooser",
		"canonical chooser remains exactly Recenter / Peek / Scan / Arrive",
		"They are **not chooser entries**",
		"Neither `Find an analog` nor a lens adds a chooser entry",
		"`Use a lens`",
		"3. `Back`",
		"4. `Back`",
		"3. `Find an analog`",
	} {
		if strings.Contains(picker, contradictory) {
			t.Errorf("nn-navigate retains contradictory picker rule %q", contradictory)
		}
	}
}

func TestNNNavigateClarifiesDismissalBackAndFullFrameHierarchy(t *testing.T) {
	data, err := os.ReadFile("../../../skills/nn-navigate/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-navigate: %v", err)
	}
	content := string(data)
	for _, required := range []string{
		"Esc at Quick actions closes only the picker; focus and graph history remain retained.",
		"Esc means parent menu; conversational `Back` means previous graph frame.",
		"### Full-frame visual hierarchy",
		"1. **Identity and action**",
		"2. **Relationship meaning**",
		"3. **Evidence and state effect**",
		"This hierarchy changes emphasis, not content",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("dismissal/back/hierarchy contract missing %q", required)
		}
	}
}

func TestNNNavigateRetainsConversationScopedMenuUIState(t *testing.T) {
	data, err := os.ReadFile("../../../skills/nn-navigate/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-navigate: %v", err)
	}
	content := string(data)
	for _, required := range []string{
		"current menu and menu stack",
		"Quick actions, All actions, Recenter destinations, Peek, Scan, and Lenses",
		"conversation-scoped UI state",
		"not notebook state",
		"compaction handoff MUST include",
		"If menu UI state is absent after compaction, it is unknown",
		"`<short-id> · Quick actions › ...`",
		"Every submenu shows this breadcrumb",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("menu UI state contract missing %q", required)
		}
	}
}

func TestNNNavigateSupportsDirectConversationalActionsAndDeterministicReturns(t *testing.T) {
	data, err := os.ReadFile("../../../skills/nn-navigate/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-navigate: %v", err)
	}
	content := string(data)
	for _, required := range []string{
		"show, explain, analogize, find an analog, visualize, quiz, scan, and arrive",
		"direct conversational intents from any menu",
		"not `nn` subcommands",
		"A Lens invoked from Lenses returns to Lenses.",
		"Show verbatim and Explain in depth return to Peek.",
		"Local territory and Global landscape return to Scan.",
		"A promoted top-level transient returns to Quick actions.",
		"Quiz suspends the picker while unanswered",
		"returns to the Lenses or Quick actions menu that invoked it",
		"`navigate` aborts Quiz and returns to Quick actions.",
		"Recenter, Teleport, Back, Forward, and Go to",
		"rerun Orient and reset the menu to Quick actions",
		"`navigate` always reruns Orient and resets to Quick actions.",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("direct action/return contract missing %q", required)
		}
	}
}

func TestNNNavigateAvoidsRedundantTransientRenders(t *testing.T) {
	data, err := os.ReadFile("../../../skills/nn-navigate/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-navigate: %v", err)
	}
	content := string(data)
	for _, required := range []string{
		"Do not redundantly rerender Focus + Map + Moves",
		"complete retained frame and notebook are unchanged",
		"compact breadcrumb, state that focus is unchanged, and reopen the invoking picker",
		"focus, filters, traversal context, or notebook changed",
		"navigation resumes after discussion",
		"explicit Refresh",
		"cached frame is stale or unknown",
		"A full render preserves the readable full map",
		"IDs, readable titles, and substantive body-derived reasons",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("transient rendering contract missing %q", required)
		}
	}
	for _, contradictory := range []string{
		"after each result, automatically restore the complete retained navigation frame, re-run Orient, render Focus + Map + Moves",
		"Automatic returns are full descriptive returns, never skeletal returns.",
		"After Show verbatim, Explain in depth, Analogize, Visualize, or a completed Quiz, the picker is already reopened.",
	} {
		if strings.Contains(content, contradictory) {
			t.Errorf("nn-navigate retains contradictory transient rerender rule %q", contradictory)
		}
	}
}

func TestNNNavigatePromotesOnlyEvidenceBackedContextualShortcuts(t *testing.T) {
	data, err := os.ReadFile("../../../skills/nn-navigate/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-navigate: %v", err)
	}
	content := string(data)
	for _, required := range []string{
		"Analog candidate survives correspondence mapping, where it holds, and where it breaks.",
		"Visualize when the evidence has process, state, or relationship structure.",
		"Quiz when the sources support a consequential distinction.",
		"Show verbatim when exact wording matters.",
		"Explain in depth when the focus is connected, contested, complex, or load-bearing.",
		"Recenter when a specific destination clearly advances the retained goal.",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("contextual shortcut criteria missing exact rule %q", required)
		}
	}
}

func TestNNNavigateOwnsTransientActionReturnAndDeepPeekContracts(t *testing.T) {
	data, err := os.ReadFile("../../../skills/nn-navigate/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-navigate: %v", err)
	}
	content := string(data)
	for _, required := range []string{
		"Show verbatim, Explain in depth, Analogize, Find an analog, and Visualize are transient actions",
		"A completed Quiz follows the same return",
		"show the compact breadcrumb, say focus is unchanged",
		"reopen Peek, Lenses, Scan, or Quick actions as specified",
		"Show the complete stored body verbatim, without truncation",
		"separate metadata, stored links, and display-only injected material",
		"source-bounded full treatment",
		"claims, details, relationships, implications, and uncertainty",
		"separate direct quotation, stored-edge evidence, and interpretation",
		"suspends the picker until the human answers, passes, skips, says `I don't know`, or says `navigate`",
		"interruption and early escape",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("transient/deep Peek contract missing %q", required)
		}
	}
}

func TestNNNavigateOwnsFindAnAnalogAndLensContracts(t *testing.T) {
	data, err := os.ReadFile("../../../skills/nn-navigate/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-navigate: %v", err)
	}
	content := string(data)
	assertions := map[string][]string{
		"moves versus lenses": {
			"## Moves versus lenses",
			"Find an analog",
			"Scan retrieval move across another region",
			"relational structure, not lexical similarity",
			"may be reached through Lenses or promoted",
			"never mutate focus or navigation history",
		},
		"analog retrieval workflow": {
			"`nn clusters`", "`nn list --search`", "`nn list --similar`", "graph neighborhoods",
			"compare relational shape",
			"correspondence mapping",
			"where the analogy holds",
			"where it breaks",
			"missing-edge suggestion",
			"comparison-only",
			"Preserve the retained focus until an explicit Recenter",
		},
		"analogize lens": {
			"### Analogize",
			"familiar analogy",
			"correspondence",
			"what it clarifies",
			"where it breaks",
			"generated, non-evidence",
		},
		"visualize lens": {
			"### Visualize",
			"spatializes meaning",
			"retained visited evidence",
			"stored Map/graph",
			"derived layout",
			"derived arrows",
			"non-stored",
			"ASCII",
			"Pi-supported Mermaid",
			"`graph`", "`flowchart`", "`stateDiagram`", "`stateDiagram-v2`", "`classDiagram`", "`erDiagram`", "`sequenceDiagram`",
			"exclude `pie`, `quadrantChart`, and `mindmap`",
		},
		"quiz lens": {
			"### Quiz",
			"source-grounded",
			"consequential concepts",
			"explicit purpose and stopping condition",
			"one question, then wait for a human Predict turn before the reveal",
			"compare the prediction",
			"note IDs and stored edge evidence",
			"misconception and why",
			"pass", "skip", "I don't know",
			"bounded to 1–3 concepts",
			"Do not invent questions",
			"derived-framework recall",
		},
		"later mutation boundary": {
			"Lens findings may inform a later explicit Recenter or link suggestion",
			"the lens itself does not mutate focus, history, notes, or links",
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

func TestNNNavigateOwnsUniversalNavigateEscapeAndResume(t *testing.T) {
	data, err := os.ReadFile("../../../skills/nn-navigate/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-navigate: %v", err)
	}
	content := string(data)

	for _, required := range []string{
		"Say navigate to reopen Focus + Map + Moves at the retained focus.",
		"after every Arrive report",
		"when exiting an extended navigation discussion",
		"Do not use this footer as a substitute for the deterministic transient-action return.",
		"reopen the invoking picker with the compact or full presentation required by the rendering policy",
		"Answer, pass/skip/I don't know, or say navigate to leave the Quiz lens and return to navigation.",
		"`navigate` during an unanswered Quiz aborts the current item",
		"without grading, revealing the answer, or forcing Quiz completion",
		"does not mutate focus or navigation history",
		"restore the complete retained navigation frame",
		"re-run Orient",
		"reset to Quick actions",
		"conversational shortcut, not an `nn` subcommand",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("nn-navigate universal navigate escape/resume contract missing %q", required)
		}
	}
}

func TestNNNavigateEscapeResumeDetailDoesNotLeakIntoDispatchReferences(t *testing.T) {
	guideData, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-guide: %v", err)
	}
	_, execute := setupNotebook(t)
	virtual, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("show virtual-nn-cli-reference: %v", err)
	}

	for name, content := range map[string]string{
		"nn-guide":                 string(guideData),
		"virtual-nn-cli-reference": virtual,
	} {
		for _, ownerOnly := range []string{
			"Say navigate to reopen Focus + Map + Moves at the retained focus.",
			"Answer, pass/skip/I don't know, or say navigate to leave the Quiz lens and return to navigation.",
			"`navigate` during an unanswered Quiz aborts the current item",
			"without grading, revealing the answer, or forcing Quiz completion",
			"restore the complete retained navigation frame",
			"invoke the adaptive quick-actions picker when the harness supports it",
		} {
			if strings.Contains(content, ownerOnly) {
				t.Errorf("%s duplicates nn-navigate navigate escape/resume detail %q", name, ownerOnly)
			}
		}
	}
}

func TestAdaptivePickerDetailDoesNotLeakIntoDispatchReferences(t *testing.T) {
	guideData, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-guide: %v", err)
	}
	_, execute := setupNotebook(t)
	virtual, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("show virtual-nn-cli-reference: %v", err)
	}

	for name, content := range map[string]string{
		"nn-guide":                 string(guideData),
		"virtual-nn-cli-reference": virtual,
	} {
		for _, ownerOnly := range []string{
			"## Adaptive hierarchical quick-actions picker",
			"up to two evidence-backed contextual concrete shortcuts",
			"All navigation actions…",
			"Lenses…",
			"A specific promoted `Lens` action may occupy a contextual shortcut slot",
			"Show verbatim",
			"Explain in depth",
			"Local territory",
			"Global landscape",
			"source-bounded full treatment",
			"suspends the picker until the human answers",
		} {
			if strings.Contains(content, ownerOnly) {
				t.Errorf("%s duplicates nn-navigate adaptive-picker detail %q", name, ownerOnly)
			}
		}
	}
}

func TestConsolidatedMenuUXIsSingleSourcedInNNNavigate(t *testing.T) {
	ownerData, err := os.ReadFile("../../../skills/nn-navigate/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-navigate: %v", err)
	}
	owner := string(ownerData)
	for _, unique := range []string{
		"### Stable menus, breadcrumbs, and effects",
		"### Direct intents and menu return transitions",
		"### Transient rendering policy",
	} {
		if got := strings.Count(owner, unique); got != 1 {
			t.Errorf("nn-navigate contract %q occurs %d times, want exactly once", unique, got)
		}
	}

	guideData, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-guide: %v", err)
	}
	_, execute := setupNotebook(t)
	virtual, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("show virtual-nn-cli-reference: %v", err)
	}
	for name, content := range map[string]string{
		"nn-guide":                 string(guideData),
		"virtual-nn-cli-reference": virtual,
	} {
		for _, ownerOnly := range []string{
			"current menu and menu stack",
			"<short-id> · Quick actions › ...",
			"→ focus changes; ○ focus retained; ■ stops; ↗ explores beyond local",
			"Never render an explicit `Back` row.",
			"Do not redundantly rerender Focus + Map + Moves",
		} {
			if strings.Contains(content, ownerOnly) {
				t.Errorf("%s duplicates consolidated nn-navigate menu UX %q", name, ownerOnly)
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
		"nn-navigate owns Find an analog and the optional Analogize, Visualize, and Quiz lenses",
	} {
		if !strings.Contains(strings.ToLower(content), strings.ToLower(required)) {
			t.Errorf("nn-guide navigation dispatch missing %q", required)
		}
	}

	for _, ownerOnly := range []string{
		"## Human-driven navigation invariant",
		"## Adaptive hierarchical quick-actions picker",
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
		"Scan retrieval move across another region",
		"correspondence mapping",
		"generated, non-evidence",
		"Pi-supported Mermaid",
		"one question, then wait for a human Predict turn before the reveal",
		"missing-edge suggestion",
		"derived-framework recall",
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

func TestNNNavigateEnforcesSemanticDirectionInEveryHumanFacingUse(t *testing.T) {
	data, err := os.ReadFile("../../../skills/nn-navigate/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-navigate: %v", err)
	}
	content := string(data)

	for _, required := range []string{
		"every human-facing directional map label, move, chooser label and description, recommendation, Peek return, and Recenter return",
		"stable emoji marker, zone name, and local relationship meaning",
		"`🔵 TOP — what the focus answers to`",
		"`🟢 BOTTOM — what builds on the focus`",
		"`🔴 LEFT — what contests or questions the focus`",
		"`🔷 RIGHT — lateral provenance or task relationships`",
		"Geometry words such as `upward`, `downward`, `left`, `right`, `above`, and `below` may supplement this semantic triple but never suffice alone.",
		"A note title or ID cannot replace the local relationship meaning.",
		"Empty zones still require the marker, zone name, and meaning",
		"Recenter 🔵 TOP — what this focus answers to: move to <id> — <readable target title> because <substantive body-derived reason>",
		"Peek 🔴 LEFT — what contests or questions this focus: inspect <id> — <readable target title> because <substantive body-derived reason>",
		"Peek returns and Recenter returns MUST restate the semantic triple",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("nn-navigate semantic-direction enforcement missing %q", required)
		}
	}

	for _, obsolete := range []string{
		"Recenter ↓ checkpoint principle",
		"Peek ↑ Bar fragility",
		"→ 4302 — peek confirms it resolves the open question",
		"🔴 LEFT <tension>",
		"🔷 RIGHT <provenance>",
		`("no LEFT: nothing contests this")`,
	} {
		if strings.Contains(content, obsolete) {
			t.Errorf("nn-navigate retains direction shorthand without the required semantic triple %q", obsolete)
		}
	}
}

func TestNNNavigateSemanticDirectionDetailDoesNotLeakIntoDispatchReferences(t *testing.T) {
	guideData, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-guide: %v", err)
	}
	_, execute := setupNotebook(t)
	virtual, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("show virtual-nn-cli-reference: %v", err)
	}

	for name, content := range map[string]string{
		"nn-guide":                 string(guideData),
		"virtual-nn-cli-reference": virtual,
	} {
		for _, ownerOnly := range []string{
			"every human-facing directional map label, move, chooser label and description, recommendation, Peek return, and Recenter return",
			"Geometry words such as `upward`, `downward`, `left`, `right`, `above`, and `below`",
			"A note title or ID cannot replace the local relationship meaning.",
			"Recenter 🔵 TOP — what this focus answers to: move to <id> — <readable target title> because <substantive body-derived reason>",
			"Peek returns and Recenter returns MUST restate the semantic triple",
		} {
			if strings.Contains(content, ownerOnly) {
				t.Errorf("%s duplicates nn-navigate semantic-direction detail %q", name, ownerOnly)
			}
		}
	}
}

func TestNNNavigateRequiresCompleteVisibleNodeDescriptionsAtEverySeam(t *testing.T) {
	data, err := os.ReadFile("../../../skills/nn-navigate/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-navigate: %v", err)
	}
	content := string(data)

	for _, required := range []string{
		"Every visible non-focus node",
		"Orient, a full Peek or Lens return, Scan, Teleport landing, Back/Forward restoration, and a full `navigate` resume",
		"`<id> — <readable title> — <body-derived central claim>`",
		"scaled to that node's inbound degree",
		"IDs supplement identity and substance; they never replace either one.",
		"[ ] every visible non-focus node has ID, readable title, and body-derived claim",
		"[ ] compact labels have an immediately adjacent complete legend; no orphan or ID-only nodes",
		"Width pressure may replace map node text with compact labels only",
		"immediately adjacent complete legend",
		"every compact label to that node's ID, readable title, and body-derived central claim",
		"Orphan labels and ID-only nodes are prohibited.",
		"Empty zones are exempt from node description",
		"retain the complete semantic gloss",
		"Every concrete quick-action target MUST include its ID, readable target title",
		"substantive body- or evidence-derived reason",
		"`supporting experiment` alone is not a substantive reason",
		"A compact unchanged transient return references the complete frame already visible instead of redrawing a skeleton.",
		"**Compliant descriptive node and compact-map fallback:**",
		"**Noncompliant skeletal presentation:**",
		"Replay-safe checkpointing",
		"the body says failed replays restore the last durable boundary",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("nn-navigate complete visible-node presentation contract missing %q", required)
		}
	}
}

func TestNNNavigateVisibleNodeDetailDoesNotLeakIntoDispatchReferences(t *testing.T) {
	guideData, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-guide: %v", err)
	}
	_, execute := setupNotebook(t)
	virtual, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("show virtual-nn-cli-reference: %v", err)
	}

	for name, content := range map[string]string{
		"nn-guide":                 string(guideData),
		"virtual-nn-cli-reference": virtual,
	} {
		for _, ownerOnly := range []string{
			"Every visible non-focus node",
			"`<id> — <readable title> — <body-derived central claim>`",
			"Width pressure may replace map node text with compact labels only",
			"Orphan labels and ID-only nodes are prohibited.",
			"`supporting experiment` alone is not a substantive reason",
			"A compact unchanged transient return references the complete frame already visible instead of redrawing a skeleton.",
			"**Compliant descriptive node and compact-map fallback:**",
			"**Noncompliant skeletal presentation:**",
		} {
			if strings.Contains(content, ownerOnly) {
				t.Errorf("%s duplicates nn-navigate visible-node presentation detail %q", name, ownerOnly)
			}
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
		"nn-navigate owns Find an analog and the optional Analogize, Visualize, and Quiz lenses",
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
		"Scan retrieval move across another region",
		"correspondence mapping",
		"generated, non-evidence",
		"Pi-supported Mermaid",
		"one question, then wait for a human Predict turn before the reveal",
		"missing-edge suggestion",
		"derived-framework recall",
	} {
		if strings.Contains(out, ownerOnly) {
			t.Errorf("virtual CLI reference duplicates nn-navigate contract %q", ownerOnly)
		}
	}
}
