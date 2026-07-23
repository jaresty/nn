package cmd

import (
	"strings"
	"testing"
)

// Assertion: TestTodoDoneFlipsOpenCheckbox — nn todo done marks first matching open checkbox as done
func TestTodoDoneFlipsOpenCheckbox(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("new", "--title", "Task note", "--type", "observation", "--content", "- [ ] write tests\n- [ ] review PR", "--no-edit")
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	id := strings.TrimPrefix(strings.TrimSpace(out), "created ")
	id = strings.Fields(id)[0]

	since := noteModified(t, execute, id)
	_, err = execute("todo", "done", id, "write tests")
	if err != nil {
		t.Fatalf("nn todo done: %v", err)
	}
	_ = since

	body := noteBody(t, execute, id)
	if !strings.Contains(body, "- [x] write tests") {
		t.Errorf("expected '- [x] write tests' in body, got:\n%s", body)
	}
	if !strings.Contains(body, "- [ ] review PR") {
		t.Errorf("expected '- [ ] review PR' still open in body, got:\n%s", body)
	}
}

// Assertion: TestTodoDoneErrorsWhenNoMatch — nn todo done errors when no open checkbox matches pattern
func TestTodoDoneErrorsWhenNoMatch(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("new", "--title", "Task note", "--type", "observation", "--content", "- [ ] write tests", "--no-edit")
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	id := strings.TrimPrefix(strings.TrimSpace(out), "created ")
	id = strings.Fields(id)[0]

	_, err = execute("todo", "done", id, "nonexistent pattern")
	if err == nil {
		t.Fatal("expected error when no matching open checkbox, got nil")
	}
}

// Assertion: TestTodoReopenFlipsDoneCheckbox — nn todo reopen marks first matching done checkbox as open
func TestTodoReopenFlipsDoneCheckbox(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("new", "--title", "Task note", "--type", "observation", "--content", "- [x] write tests\n- [x] review PR", "--no-edit")
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	id := strings.TrimPrefix(strings.TrimSpace(out), "created ")
	id = strings.Fields(id)[0]

	_, err = execute("todo", "reopen", id, "write tests")
	if err != nil {
		t.Fatalf("nn todo reopen: %v", err)
	}

	body := noteBody(t, execute, id)
	if !strings.Contains(body, "- [ ] write tests") {
		t.Errorf("expected '- [ ] write tests' in body, got:\n%s", body)
	}
	if !strings.Contains(body, "- [x] review PR") {
		t.Errorf("expected '- [x] review PR' still done in body, got:\n%s", body)
	}
}

// Assertion: TestTodoReopenErrorsWhenNoMatch — nn todo reopen errors when no done checkbox matches pattern
func TestTodoReopenErrorsWhenNoMatch(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("new", "--title", "Task note", "--type", "observation", "--content", "- [x] write tests", "--no-edit")
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	id := strings.TrimPrefix(strings.TrimSpace(out), "created ")
	id = strings.Fields(id)[0]

	_, err = execute("todo", "reopen", id, "nonexistent pattern")
	if err == nil {
		t.Fatal("expected error when no matching done checkbox, got nil")
	}
}

// Assertion: TestTodoDoneCaseInsensitive — nn todo done pattern match is case-insensitive
func TestTodoDoneCaseInsensitive(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("new", "--title", "Task note", "--type", "observation", "--content", "- [ ] Write Tests", "--no-edit")
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	id := strings.TrimPrefix(strings.TrimSpace(out), "created ")
	id = strings.Fields(id)[0]

	_, err = execute("todo", "done", id, "write tests")
	if err != nil {
		t.Fatalf("nn todo done (case-insensitive): %v", err)
	}

	body := noteBody(t, execute, id)
	if !strings.Contains(body, "- [x] Write Tests") {
		t.Errorf("expected '- [x] Write Tests' in body, got:\n%s", body)
	}
}

// Assertion: TestTodoListGroupedByNote — nn todo list groups open checkboxes under note ID + title header
func TestTodoListGroupedByNote(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("new", "--title", "Alpha tasks", "--type", "observation", "--content", "- [ ] first item\n- [x] done item\n- [ ] second item", "--no-edit")
	if err != nil {
		t.Fatalf("nn new alpha: %v", err)
	}
	idA := strings.Fields(strings.TrimPrefix(strings.TrimSpace(out), "created "))[0]

	out, err = execute("new", "--title", "Beta tasks", "--type", "observation", "--content", "- [ ] beta item", "--no-edit")
	if err != nil {
		t.Fatalf("nn new beta: %v", err)
	}
	idB := strings.Fields(strings.TrimPrefix(strings.TrimSpace(out), "created "))[0]

	out, err = execute("todo", "list")
	if err != nil {
		t.Fatalf("nn todo list: %v", err)
	}

	if !strings.Contains(out, idA) {
		t.Errorf("expected note ID %s in output", idA)
	}
	if !strings.Contains(out, "Alpha tasks") {
		t.Errorf("expected title 'Alpha tasks' in output")
	}
	if !strings.Contains(out, "- [ ] first item") {
		t.Errorf("expected '- [ ] first item' in output")
	}
	if !strings.Contains(out, "- [ ] second item") {
		t.Errorf("expected '- [ ] second item' in output")
	}
	if strings.Contains(out, "- [x] done item") {
		t.Errorf("expected done item to be excluded from output")
	}
	if !strings.Contains(out, idB) {
		t.Errorf("expected note ID %s in output", idB)
	}
	if !strings.Contains(out, "Beta tasks") {
		t.Errorf("expected title 'Beta tasks' in output")
	}
	if !strings.Contains(out, "- [ ] beta item") {
		t.Errorf("expected '- [ ] beta item' in output")
	}
}

// Assertion: TestTodoListExcludesNotesWithNoOpenItems — nn todo list omits notes with no open checkboxes
func TestTodoListExcludesNotesWithNoOpenItems(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("new", "--title", "All done", "--type", "observation", "--content", "- [x] finished", "--no-edit")
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	id := strings.Fields(strings.TrimPrefix(strings.TrimSpace(out), "created "))[0]

	out, err = execute("todo", "list")
	if err != nil {
		t.Fatalf("nn todo list: %v", err)
	}
	if strings.Contains(out, id) {
		t.Errorf("expected note with no open items to be excluded, got: %s", out)
	}
}

// Assertion: TestTodoListDefaultExcludesBlockedNotes — nn todo list default excludes notes blocked by incomplete requires target
func TestTodoListDefaultExcludesBlockedNotes(t *testing.T) {
	_, execute := setupNotebook(t)

	// prereq note with open item — blocking
	out, err := execute("new", "--title", "Prereq", "--type", "observation", "--content", "- [ ] incomplete prereq", "--no-edit")
	if err != nil {
		t.Fatalf("nn new prereq: %v", err)
	}
	prereqID := strings.Fields(strings.TrimPrefix(strings.TrimSpace(out), "created "))[0]

	// blocked note that requires prereq
	out, err = execute("new", "--title", "Blocked task", "--type", "observation", "--content", "- [ ] blocked item", "--no-edit")
	if err != nil {
		t.Fatalf("nn new blocked: %v", err)
	}
	blockedID := strings.Fields(strings.TrimPrefix(strings.TrimSpace(out), "created "))[0]

	_, err = execute("link", blockedID, prereqID, "--type", "requires", "--annotation", "needs prereq first")
	if err != nil {
		t.Fatalf("nn link: %v", err)
	}

	out, err = execute("todo", "list")
	if err != nil {
		t.Fatalf("nn todo list: %v", err)
	}

	// prereq itself has open items and no requires links — should appear
	if !strings.Contains(out, prereqID) {
		t.Errorf("expected prereq note %s to appear (it has open items, not blocked)", prereqID)
	}
	// blocked note should be excluded by default
	if strings.Contains(out, blockedID) {
		t.Errorf("expected blocked note %s to be excluded by default", blockedID)
	}
}

// Assertion: TestTodoListAllShowsBlockedNotes — nn todo list --all shows blocked notes too
func TestTodoListAllShowsBlockedNotes(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("new", "--title", "Prereq", "--type", "observation", "--content", "- [ ] incomplete prereq", "--no-edit")
	if err != nil {
		t.Fatalf("nn new prereq: %v", err)
	}
	prereqID := strings.Fields(strings.TrimPrefix(strings.TrimSpace(out), "created "))[0]

	out, err = execute("new", "--title", "Blocked task", "--type", "observation", "--content", "- [ ] blocked item", "--no-edit")
	if err != nil {
		t.Fatalf("nn new blocked: %v", err)
	}
	blockedID := strings.Fields(strings.TrimPrefix(strings.TrimSpace(out), "created "))[0]

	_, err = execute("link", blockedID, prereqID, "--type", "requires", "--annotation", "needs prereq first")
	if err != nil {
		t.Fatalf("nn link: %v", err)
	}

	out, err = execute("todo", "list", "--all")
	if err != nil {
		t.Fatalf("nn todo list --all: %v", err)
	}

	if !strings.Contains(out, prereqID) {
		t.Errorf("expected prereq note %s in --all output", prereqID)
	}
	if !strings.Contains(out, blockedID) {
		t.Errorf("expected blocked note %s in --all output", blockedID)
	}
}

// Assertion: TestTodoListDefaultShowsNoteWithNoRequiresLinks — nn todo list default shows notes with open items and no requires links
func TestTodoListDefaultShowsNoteWithNoRequiresLinks(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("new", "--title", "Free task", "--type", "observation", "--content", "- [ ] free item", "--no-edit")
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	id := strings.Fields(strings.TrimPrefix(strings.TrimSpace(out), "created "))[0]

	out, err = execute("todo", "list")
	if err != nil {
		t.Fatalf("nn todo list: %v", err)
	}
	if !strings.Contains(out, id) {
		t.Errorf("expected note %s with no requires links to appear in default output", id)
	}
}

// Assertion: TestTodoListDefaultExcludesWaitingItems — nn todo list default excludes items tagged [waiting: reason]
func TestTodoListDefaultExcludesWaitingItems(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("new", "--title", "Waiting task", "--type", "observation", "--content", "- [ ] [waiting: Josh to review] submit the PR\n- [ ] unblocked item", "--no-edit")
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	id := strings.Fields(strings.TrimPrefix(strings.TrimSpace(out), "created "))[0]

	out, err = execute("todo", "list")
	if err != nil {
		t.Fatalf("nn todo list: %v", err)
	}
	if strings.Contains(out, "submit the PR") {
		t.Errorf("expected waiting item to be excluded from default output, got:\n%s", out)
	}
	if !strings.Contains(out, id) {
		t.Errorf("expected note %s to appear (has unblocked item)", id)
	}
	if !strings.Contains(out, "unblocked item") {
		t.Errorf("expected unblocked item to appear in output")
	}
}

// Assertion: TestTodoListWaitingFlagShowsWaitingItems — nn todo list --waiting shows waiting items with reason
func TestTodoListWaitingFlagShowsWaitingItems(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("new", "--title", "Waiting task", "--type", "observation", "--content", "- [ ] [waiting: Josh to review] submit the PR", "--no-edit")
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	id := strings.Fields(strings.TrimPrefix(strings.TrimSpace(out), "created "))[0]

	out, err = execute("todo", "list", "--waiting")
	if err != nil {
		t.Fatalf("nn todo list --waiting: %v", err)
	}
	if !strings.Contains(out, id) {
		t.Errorf("expected note %s in --waiting output", id)
	}
	if !strings.Contains(out, "submit the PR") {
		t.Errorf("expected waiting item in --waiting output")
	}
	if !strings.Contains(out, "Josh to review") {
		t.Errorf("expected waiting reason 'Josh to review' in --waiting output")
	}
}

// Assertion: TestTodoListContextFilterShowsMatchingItems — nn todo list --context filters to matching @context items
func TestTodoListContextFilterShowsMatchingItems(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("new", "--title", "Context task", "--type", "observation", "--content", "- [ ] @phone call the vendor\n- [ ] @computer write the report", "--no-edit")
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	id := strings.Fields(strings.TrimPrefix(strings.TrimSpace(out), "created "))[0]

	out, err = execute("todo", "list", "--context", "phone")
	if err != nil {
		t.Fatalf("nn todo list --context phone: %v", err)
	}
	if !strings.Contains(out, id) {
		t.Errorf("expected note %s in --context phone output", id)
	}
	if !strings.Contains(out, "call the vendor") {
		t.Errorf("expected @phone item in --context phone output")
	}
	if strings.Contains(out, "write the report") {
		t.Errorf("expected @computer item excluded from --context phone output")
	}
}

func noteModified(t *testing.T, execute func(...string) (string, error), id string) string {
	t.Helper()
	out, err := execute("show", id)
	if err != nil {
		t.Fatalf("nn show %s: %v", id, err)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "modified:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "modified:"))
		}
	}
	t.Fatalf("no modified: line in nn show output for %s", id)
	return ""
}

func noteBody(t *testing.T, execute func(...string) (string, error), id string) string {
	t.Helper()
	out, err := execute("show", id)
	if err != nil {
		t.Fatalf("nn show %s: %v", id, err)
	}
	// Body starts after the YAML frontmatter separator
	parts := strings.SplitN(out, "---\n", 3)
	if len(parts) < 3 {
		t.Fatalf("unexpected nn show format for %s: %q", id, out)
	}
	return parts[2]
}
