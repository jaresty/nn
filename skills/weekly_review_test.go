package skills

import (
	"strings"
	"testing"
)

func readSkill(t *testing.T, name string) string {
	t.Helper()
	data, err := FS.ReadFile(name + "/SKILL.md")
	if err != nil {
		t.Fatalf("read %s/SKILL.md: %v", name, err)
	}
	return string(data)
}

func requireAll(t *testing.T, label, body string, phrases ...string) {
	t.Helper()
	for _, phrase := range phrases {
		if !strings.Contains(body, phrase) {
			t.Errorf("%s: expected %q; got no match", label, phrase)
		}
	}
}

// Assertion [P6]: nn-session-debrief --weekly delegates the sweep to nn-weekly-review and no longer holds the expired-sweep commands.
func TestDebriefWeeklyDelegatesToWeeklyReview(t *testing.T) {
	body := readSkill(t, "nn-session-debrief")
	requireAll(t, "Assertion [P6] debrief delegation", body, "nn-weekly-review")
	if strings.Contains(body, "nn list --expired --count") {
		t.Errorf("Assertion [P6] debrief delegation: expired-sweep command block still present in nn-session-debrief")
	}
}

// Assertion [P2]: the skill names a default per-bucket confirmation mode and an --auto mode.
func TestWeeklyReviewNamesTwoAutonomyModes(t *testing.T) {
	body := readSkill(t, "nn-weekly-review")
	requireAll(t, "Assertion [P2] autonomy modes", body, "--auto", "once per bucket")
}

// Assertion [P3]: --auto terminates on an observable count of 0, a no-progress pass, an exhausted budget, or a user stop.
func TestWeeklyReviewAutoLoopHasFourStops(t *testing.T) {
	body := readSkill(t, "nn-weekly-review")
	requireAll(t, "Assertion [P3] loop stops", body,
		"nn list --expired --count",
		"no-progress",
		"budget",
		"user says stop",
	)
}

// Assertion [P4]: guards hold in --auto: protocol and permanent notes are never deleted, backlinked notes are inlined first.
func TestWeeklyReviewGuardsHoldInAuto(t *testing.T) {
	body := readSkill(t, "nn-weekly-review")
	requireAll(t, "Assertion [P4] guards", body,
		"Never auto-delete",
		"protocol",
		"permanent",
		"inline the key claim",
	)
}

// Assertion [P5]: every run ends with a receipt of before and after counts, deleted IDs and the residual list.
func TestWeeklyReviewEndsWithReceipt(t *testing.T) {
	body := readSkill(t, "nn-weekly-review")
	requireAll(t, "Assertion [P5] receipt", body,
		"## Receipt",
		"before and after counts",
		"deleted IDs",
		"residual",
	)
}

// Assertion [P1]: the skill description leads with "Use when", matching the routing convention enforced for every skill.
func TestWeeklyReviewDescriptionLeadsWithUseWhen(t *testing.T) {
	body := readSkill(t, "nn-weekly-review")
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "description:") {
			if !strings.Contains(line, "Use when") {
				t.Errorf("Assertion [P1] description: does not lead with 'Use when': %q", line)
			}
			return
		}
	}
	t.Errorf("Assertion [P1] description: no description: line in frontmatter")
}
