package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNNNavigateHasCompactCoreAndExactlySixReferences(t *testing.T) {
	root := filepath.Join("..", "..", "..", "skills", "nn-navigate")
	core, err := os.ReadFile(filepath.Join(root, "SKILL.md"))
	if err != nil {
		t.Fatalf("read core: %v", err)
	}
	if len(core) < 15*1024 || len(core) > 25*1024 {
		t.Fatalf("nn-navigate core is %d bytes, want preferred compact range 15-25 KiB (and always under 50 KiB)", len(core))
	}
	if len(core) >= 50*1024 {
		t.Fatalf("nn-navigate core is %d bytes, must be under 50 KiB", len(core))
	}

	entries, err := os.ReadDir(filepath.Join(root, "references"))
	if err != nil {
		t.Fatalf("read references: %v", err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	want := []string{"ask.md", "lenses.md", "movement.md", "presentation.md", "scan-and-routes.md", "state.md"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("nn-navigate references = %v, want exactly %v", names, want)
	}
}

// retained properties [22]-[27]: the compact core remains a binding dispatcher;
// splitting detail must not split away state, movement, presentation, or recovery invariants.
func TestNNNavigateCompactCoreRetainsBindingDispatchContracts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "skills", "nn-navigate", "SKILL.md"))
	if err != nil {
		t.Fatalf("read core: %v", err)
	}
	core := string(data)

	properties := map[string][]string{
		"[22] owning-reference fetch and action dispatch": {
			"Before executing any applicable action, MUST fetch every owning reference",
			"nn skills get nn-navigate --reference <name>",
			"presentation", "ask", "movement", "scan-and-routes", "lenses", "state",
		},
		"[23] complete conversation-scoped state model": {
			"focus: note-id", "goal: string", "filters: {}", "back: []", "forward: []",
			"bookmarks: {}", "visited_evidence: []", "current_menu:", "menu_stack: []",
		},
		"[24] focus mutation invariants": {
			"Teleport, Visit, Recenter, and Go to may adopt a new destination",
			"Back and Forward may change focus only by restoring a retained frame",
			"No other action changes focus",
		},
		"[25] canonical Orient commands": {
			"nn graph show --focus <id> --depth 1 --direction both --zones --presentation-hints --color always --format text",
			"nn graph bodies --focus <id> --depth 1 --direction both --page 1",
			"MUST NOT make body-derived claims",
		},
		"[26] blocking presentation checklist": {
			"Navigation presentation blocker checklist",
			"adjacent evidence index preserves ID, title, type, degree, importance marker, and body-derived claim",
			"Arrive always visible as final top-level action",
			"A missing item blocks presentation",
		},
		"[27] compaction dispatch": {
			"Before compaction", "--reference state", "full current frame", "Back and Forward stacks",
			"every bookmark", "current menu and ordered menu stack", "never invent",
		},
	}
	for property, snippets := range properties {
		for _, snippet := range snippets {
			if !strings.Contains(core, snippet) {
				t.Errorf("retained property %s missing %q", property, snippet)
			}
		}
	}
}

func TestNNNavigatePresentationReferencePreservesFourRowPickerSemantics(t *testing.T) {
	path := filepath.Join("..", "..", "..", "skills", "nn-navigate", "references", "presentation.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read presentation reference: %v", err)
	}
	presentation := string(data)
	for _, required := range []string{
		"up to one evidence-backed contextual concrete shortcut",
		"stable `Lenses…` row",
		"stable `All navigation actions…` row",
		"final row is always `■ Arrive`",
	} {
		if !strings.Contains(presentation, required) {
			t.Errorf("presentation reference missing four-row picker rule %q", required)
		}
	}
	if strings.Contains(presentation, "up to two justified concrete shortcuts") {
		t.Fatal("presentation reference contradicts the one-shortcut/final-Arrive picker contract")
	}

	start := strings.Index(presentation, "**Compliant top-level picker:**")
	end := strings.Index(presentation, "Relationship templates embedded")
	if start < 0 || end <= start {
		t.Fatal("compliant top-level picker example is missing")
	}
	example := presentation[start:end]
	if !strings.Contains(example, "- ■ Arrive") {
		t.Fatal("compliant top-level picker example omits the mandatory final Arrive row")
	}
}

// retained property [1a]: a completed lens selected from a Guided picker
// visibly returns to its invoking menu rather than taking the quiet-return path.
func TestNNNavigatePresentationReferenceReturnsPickerSelectedLensesToInvokingMenu(t *testing.T) {
	path := filepath.Join("..", "..", "..", "skills", "nn-navigate", "references", "presentation.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read presentation reference: %v", err)
	}
	presentation := string(data)
	const assertion = "[P1a] Guided picker-selected lens completion reopens the invoking menu"
	const required = "A completed action selected from a Guided picker always reopens its invoking menu"
	if !strings.Contains(presentation, required) {
		t.Errorf("%s: presentation reference missing %q", assertion, required)
	}
}

// retained property [1b]: quiet return remains available only to a completed
// lens requested directly in conversation, not to a picker selection.
func TestNNNavigatePresentationReferenceLimitsQuietReturnToConversationalIntent(t *testing.T) {
	path := filepath.Join("..", "..", "..", "skills", "nn-navigate", "references", "presentation.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read presentation reference: %v", err)
	}
	presentation := string(data)
	const assertion = "[P1b] quiet return is limited to conversational direct intent"
	const required = "Only a completed action requested directly in conversation may use a quiet return"
	if !strings.Contains(presentation, required) {
		t.Errorf("%s: presentation reference missing %q", assertion, required)
	}
}

// retained property [2a]: every submenu makes its Esc dismissal affordance visible.
func TestNNNavigatePresentationReferenceMakesSubmenuEscapeVisible(t *testing.T) {
	path := filepath.Join("..", "..", "..", "skills", "nn-navigate", "references", "presentation.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read presentation reference: %v", err)
	}
	presentation := string(data)
	const assertion = "[P2a] every submenu visibly identifies Esc"
	const required = "Every submenu picker visibly displays `Esc`"
	if !strings.Contains(presentation, required) {
		t.Errorf("%s: presentation reference missing %q", assertion, required)
	}
}

// retained property [2b]: every submenu's visible Esc guidance names its parent.
func TestNNNavigatePresentationReferenceNamesSubmenuEscapeDestination(t *testing.T) {
	path := filepath.Join("..", "..", "..", "skills", "nn-navigate", "references", "presentation.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read presentation reference: %v", err)
	}
	presentation := string(data)
	const assertion = "[P2b] every submenu Esc hint names its parent menu"
	const required = "The guidance names that submenu's parent menu"
	if !strings.Contains(presentation, required) {
		t.Errorf("%s: presentation reference missing %q", assertion, required)
	}
}

// retained property [3a]: visible Esc guidance is not an option and preserves
// the canonical rows in every submenu.
func TestNNNavigatePresentationReferencePreservesSubmenuRowsWithEscapeGuidance(t *testing.T) {
	path := filepath.Join("..", "..", "..", "skills", "nn-navigate", "references", "presentation.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read presentation reference: %v", err)
	}
	presentation := string(data)
	const assertion = "[P3a] Esc guidance preserves canonical submenu rows"
	const required = "canonical submenu rows remain unchanged"
	if !strings.Contains(presentation, required) {
		t.Errorf("%s: presentation reference missing %q", assertion, required)
	}
}

// retained property [3b]: visible Esc guidance never creates a fifth picker row.
func TestNNNavigatePresentationReferenceKeepsEscapeGuidanceOutsidePickerRows(t *testing.T) {
	path := filepath.Join("..", "..", "..", "skills", "nn-navigate", "references", "presentation.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read presentation reference: %v", err)
	}
	presentation := string(data)
	const assertion = "[P3b] Esc guidance keeps every submenu at four rows or fewer"
	const required = "each submenu picker still contains at most four rows"
	if !strings.Contains(presentation, required) {
		t.Errorf("%s: presentation reference missing %q", assertion, required)
	}
}

func TestNNNavigatePresentationReferencePreservesLosslessImportanceLayout(t *testing.T) {
	path := filepath.Join("..", "..", "..", "skills", "nn-navigate", "references", "presentation.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read presentation reference: %v", err)
	}
	presentation := string(data)

	for property, required := range map[string][]string{
		"four-zone geometry and visible emptiness": {
			"Every empty zone visibly occupies its geometric slot as `[∅]`",
			"TOP is above focus, LEFT is left, RIGHT is right, and BOTTOM is below",
		},
		"view-local importance separate from connectivity": {
			"view-local importance",
			"at most two `★` decision-shaping nodes",
			"`◆` decision-supporting",
			"`·` orienting context",
			"Degree remains connectivity",
		},
		"direct geometry and secondary ledger": {
			"Direct focus relationships appear in the geometry",
			"secondary stored-relationship ledger",
			"arrowhead beside its stored target",
			"reciprocal vertical relationships",
		},
	} {
		for _, snippet := range required {
			if !strings.Contains(presentation, snippet) {
				t.Errorf("presentation property %s missing %q", property, snippet)
			}
		}
	}
}

// Retained properties [1]-[5]: vertical zones share one post-classification
// renderer without changing stored edge meaning or the surrounding evidence hierarchy.
func TestNNNavigatePresentationReferenceDefinesSharedVerticalZoneRenderer(t *testing.T) {
	path := filepath.Join("..", "..", "..", "skills", "nn-navigate", "references", "presentation.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read presentation reference: %v", err)
	}
	presentation := string(data)

	properties := map[string][]string{
		"[1] stored edge fidelity survives placement": {
			"one post-classification vertical renderer",
			"stored source, canonical type, and stored target remain invariant",
			"arrowhead beside its stored target",
		},
		"[2] exact edge sets select the rendering mode": {
			"exactly one stored edge uses an attached typed stem",
			"exactly one reciprocal pair over one focus/node pair",
			"centered borderless attached rail",
		},
		"[3] dense geometry preserves one-to-one correspondence": {
			"one endpoint-complete unit per stored edge",
			"rail remains attached to the vertical-zone axis",
			"[FOCUS] ──refines──────→ ★ [B3]",
		},
		"[4] narrow wrapping keeps target arrows attached": {
			"wrap only inside that edge unit",
			"arrow and stored target remain on the same line",
			"→ ◆ [B2]",
		},
		"[5] surrounding evidence hierarchy remains intact": {
			"Direct focus relationships remain in geometry",
			"neighbor-to-neighbor stored relationships remain in the secondary stored-relationship ledger",
			"evidence index still maps each note label exactly once",
		},
	}
	for property, snippets := range properties {
		for _, snippet := range snippets {
			if !strings.Contains(presentation, snippet) {
				t.Errorf("vertical renderer property %s missing %q", property, snippet)
			}
		}
	}

	for _, obsolete := range []string{
		"separate TOP and BOTTOM rendering rules",
		"derive arrow direction from placement",
		"detached rectangular comb",
	} {
		if strings.Contains(presentation, obsolete) {
			t.Errorf("vertical renderer retains rejected rule %q", obsolete)
		}
	}
}

func TestNNNavigateReferenceActionOwnership(t *testing.T) {
	root := filepath.Join("..", "..", "..", "skills", "nn-navigate")
	files := map[string]string{}
	core, err := os.ReadFile(filepath.Join(root, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	files["core"] = string(core)
	for _, name := range []string{"ask", "lenses", "movement", "presentation", "scan-and-routes", "state"} {
		data, err := os.ReadFile(filepath.Join(root, "references", name+".md"))
		if err != nil {
			t.Fatalf("read %s reference: %v", name, err)
		}
		files[name] = string(data)
	}

	owners := map[string]string{
		"Positioned → AskPrepared → AwaitingHuman → ResultAvailable → Positioned": "ask",
		"Generate a **familiar analogy**":                                         "lenses",
		"**Selection completes the landing decision.**":                           "movement",
		"##### Stable emoji relay palette":                                        "presentation",
		"nn graph routes --focus ID --links TYPES --search QUERY":                 "scan-and-routes",
		"an existing exact-case name requires explicit confirmation":              "state",
	}
	for marker, owner := range owners {
		for file, content := range files {
			got := strings.Contains(content, marker)
			if file == owner && !got {
				t.Errorf("owner %s missing action contract %q", owner, marker)
			}
			if file != owner && got {
				t.Errorf("action contract %q duplicated in %s; owner is %s", marker, file, owner)
			}
		}
	}
}

func TestNNNavigateBundlePreservesKeyContracts(t *testing.T) {
	root := filepath.Join("..", "..", "..", "skills", "nn-navigate")
	var bundle strings.Builder
	for _, path := range []string{
		"SKILL.md", "references/presentation.md", "references/ask.md", "references/movement.md",
		"references/scan-and-routes.md", "references/lenses.md", "references/state.md",
	} {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		bundle.Write(data)
		bundle.WriteByte('\n')
	}
	all := bundle.String()
	for _, required := range []string{
		"Focus → Map → Moves",
		"Recenter — move focus",
		"Peek — inspect without moving",
		"Scan — zoom out without moving",
		"◇ Ask…",
		"■ Arrive",
		"🔵 TOP — what the focus answers to",
		"Illustrative layout and inferred relationships — not literal notebook structure.",
		"reopen graph ask",
		"Find an analog",
		"Say navigate to reopen Focus + Map + Moves at the retained focus.",
		"This state lasts only for the conversation.",
	} {
		if !strings.Contains(all, required) {
			t.Errorf("split bundle lost key contract %q", required)
		}
	}
}
