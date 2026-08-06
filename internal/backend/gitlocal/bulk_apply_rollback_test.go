package gitlocal_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/backend/gitlocal"
	"github.com/jaresty/nn/internal/note"
)

type bulkApplyPathState struct {
	Mode os.FileMode
	Data string
	Link string
}

type bulkApplyRepoState struct {
	Head     string
	HeadRef  string
	Refs     string
	Index    bulkApplyPathState
	Worktree map[string]bulkApplyPathState
	Config   map[string]bulkApplyPathState
	Extra    map[string]bulkApplyPathState
	InputIDs []string
}

func mustBulkApplyTest(t *testing.T, label string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", label, err)
	}
}

func bulkApplyGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	mustBulkApplyTest(t, "git "+strings.Join(args, " ")+" output="+string(out), err)
	return string(out)
}

func snapshotBulkApplyPath(path string) (bulkApplyPathState, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return bulkApplyPathState{}, nil
	}
	if err != nil {
		return bulkApplyPathState{}, err
	}
	state := bulkApplyPathState{Mode: info.Mode()}
	switch {
	case info.Mode().IsRegular():
		data, err := os.ReadFile(path)
		state.Data = string(data)
		return state, err
	case info.Mode()&os.ModeSymlink != 0:
		state.Link, err = os.Readlink(path)
		return state, err
	default:
		return state, nil
	}
}

func snapshotBulkApplyTree(root string, skipGit bool) (map[string]bulkApplyPathState, error) {
	states := make(map[string]bulkApplyPathState)
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if skipGit && (rel == ".git" || strings.HasPrefix(rel, ".git"+string(filepath.Separator))) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		state, err := snapshotBulkApplyPath(path)
		if err != nil {
			return err
		}
		states[rel] = state
		return nil
	})
	return states, err
}

func captureBulkApplyRepoState(t *testing.T, dir, configDir string, inputs []*note.Note) bulkApplyRepoState {
	t.Helper()
	indexPath := strings.TrimSpace(bulkApplyGit(t, dir, "rev-parse", "--git-path", "index"))
	if !filepath.IsAbs(indexPath) {
		indexPath = filepath.Join(dir, indexPath)
	}
	index, err := snapshotBulkApplyPath(indexPath)
	mustBulkApplyTest(t, "snapshot index", err)
	worktree, err := snapshotBulkApplyTree(dir, true)
	mustBulkApplyTest(t, "snapshot worktree", err)
	config, err := snapshotBulkApplyTree(configDir, false)
	mustBulkApplyTest(t, "snapshot config", err)
	extra := make(map[string]bulkApplyPathState)
	ids := make([]string, len(inputs))
	for i, n := range inputs {
		ids[i] = n.ID
		path := filepath.Join(dir, n.Filename())
		extra[path], err = snapshotBulkApplyPath(path)
		mustBulkApplyTest(t, "snapshot requested path", err)
	}
	headRefCmd := exec.Command("git", "symbolic-ref", "-q", "HEAD")
	headRefCmd.Dir = dir
	headRef, _ := headRefCmd.Output()
	return bulkApplyRepoState{
		Head:     bulkApplyGit(t, dir, "rev-parse", "HEAD"),
		HeadRef:  string(headRef),
		Refs:     bulkApplyGit(t, dir, "for-each-ref", "--format=%(refname)%00%(objectname)"),
		Index:    index,
		Worktree: worktree,
		Config:   config,
		Extra:    extra,
		InputIDs: ids,
	}
}

func diffBulkApplyRepoState(t *testing.T, dir, configDir string, inputs []*note.Note, before bulkApplyRepoState) string {
	t.Helper()
	after := captureBulkApplyRepoState(t, dir, configDir, inputs)
	var diffs []string
	if after.Head != before.Head || after.HeadRef != before.HeadRef || after.Refs != before.Refs {
		diffs = append(diffs, "HEAD or refs changed")
	}
	if !reflect.DeepEqual(after.Index, before.Index) {
		diffs = append(diffs, "raw index bytes or mode changed")
	}
	if !reflect.DeepEqual(after.Worktree, before.Worktree) {
		diffs = append(diffs, "worktree path existence/type/content/mode/link target changed")
	}
	if !reflect.DeepEqual(after.Config, before.Config) {
		diffs = append(diffs, "config or lock state changed")
	}
	if !reflect.DeepEqual(after.Extra, before.Extra) {
		diffs = append(diffs, "requested or outside path changed")
	}
	if !reflect.DeepEqual(after.InputIDs, before.InputIDs) {
		diffs = append(diffs, "caller-owned note IDs changed")
	}
	return strings.Join(diffs, "; ")
}

func TestBulkApplySucceedsOnDetachedHead(t *testing.T) {
	dir := t.TempDir()
	configDir := t.TempDir()
	initGitRepo(t, dir)
	b, err := gitlocal.NewWithConfigDir(dir, configDir)
	mustBulkApplyTest(t, "new backend", err)
	mustBulkApplyTest(t, "seed attached HEAD", b.Write(makeNote("detached seed")))
	bulkApplyGit(t, dir, "checkout", "--detach", "-q")
	before := bulkApplyGit(t, dir, "rev-list", "--count", "HEAD")

	fresh := makeNote("detached fresh")
	fresh.Body = "detached body"
	mustBulkApplyTest(t, "BulkApply detached HEAD", b.BulkApply([]*note.Note{fresh}, nil))
	if after := bulkApplyGit(t, dir, "rev-list", "--count", "HEAD"); after != "2\n" || before != "1\n" {
		t.Fatalf("detached BulkApply commit count: before=%q after=%q", before, after)
	}
	if out := bulkApplyGit(t, dir, "show", "HEAD:"+fresh.Filename()); !strings.Contains(out, "detached body") {
		t.Fatalf("detached BulkApply commit lacks note content: %s", out)
	}
	symbolic := exec.Command("git", "symbolic-ref", "-q", "HEAD")
	symbolic.Dir = dir
	if out, symbolicErr := symbolic.CombinedOutput(); symbolicErr == nil {
		t.Fatalf("BulkApply attached detached HEAD: %s", out)
	}
}

func TestBulkApplySuccessPreservesUnrelatedState(t *testing.T) {
	dir := t.TempDir()
	configDir := t.TempDir()
	initGitRepo(t, dir)
	b, err := gitlocal.NewWithConfigDir(dir, configDir)
	mustBulkApplyTest(t, "new backend", err)

	existing := makeNote("existing")
	existing.Body = "before"
	mustBulkApplyTest(t, "write existing note", b.Write(existing))
	unrelatedPath := filepath.Join(dir, "unrelated.txt")
	mustBulkApplyTest(t, "write unrelated file", os.WriteFile(unrelatedPath, []byte("staged before BulkApply"), 0o640))
	bulkApplyGit(t, dir, "add", "unrelated.txt")
	unrelatedIndexBefore := bulkApplyGit(t, dir, "ls-files", "--stage", "--", "unrelated.txt")
	unrelatedFileBefore, err := snapshotBulkApplyPath(unrelatedPath)
	mustBulkApplyTest(t, "snapshot unrelated file", err)

	fresh := makeNote("fresh")
	fresh.Body = "new"
	updated := *existing
	updated.Body = "after"
	mustBulkApplyTest(t, "BulkApply", b.BulkApply([]*note.Note{fresh}, []*note.Note{&updated}))

	if got := bulkApplyGit(t, dir, "ls-files", "--stage", "--", "unrelated.txt"); got != unrelatedIndexBefore {
		t.Fatalf("unrelated staged index entry changed: before=%q after=%q", unrelatedIndexBefore, got)
	}
	show := exec.Command("git", "show", "HEAD:unrelated.txt")
	show.Dir = dir
	if out, showErr := show.CombinedOutput(); showErr == nil {
		t.Fatalf("BulkApply commit captured unrelated staged file: %s", out)
	}
	unrelatedFileAfter, err := snapshotBulkApplyPath(unrelatedPath)
	mustBulkApplyTest(t, "snapshot unrelated file after BulkApply", err)
	if !reflect.DeepEqual(unrelatedFileAfter, unrelatedFileBefore) {
		t.Fatalf("unrelated worktree file changed: before=%#v after=%#v", unrelatedFileBefore, unrelatedFileAfter)
	}
}

func TestBulkApplyUpdateKeepsExistingPath(t *testing.T) {
	dir := t.TempDir()
	configDir := t.TempDir()
	initGitRepo(t, dir)
	b, err := gitlocal.NewWithConfigDir(dir, configDir)
	mustBulkApplyTest(t, "new backend", err)

	existing := makeNote("original title")
	existing.Body = "before"
	mustBulkApplyTest(t, "write existing note", b.Write(existing))
	originalPath := filepath.Join(dir, existing.Filename())
	updated := *existing
	updated.Title = "renamed title"
	updated.Body = "after"
	derivedRenamedPath := filepath.Join(dir, updated.Filename())
	mustBulkApplyTest(t, "BulkApply renamed update", b.BulkApply(nil, []*note.Note{&updated}))

	worktreeData, err := os.ReadFile(originalPath)
	mustBulkApplyTest(t, "read original path", err)
	headData := bulkApplyGit(t, dir, "show", "HEAD:"+existing.Filename())
	for location, data := range map[string]string{"worktree": string(worktreeData), "HEAD": headData} {
		if !strings.Contains(data, "title: renamed title") || !strings.Contains(data, "after") {
			t.Fatalf("original update path lacks renamed content in %s: %s", location, data)
		}
	}
	if derivedRenamedPath != originalPath {
		if _, statErr := os.Lstat(derivedRenamedPath); !os.IsNotExist(statErr) {
			t.Fatalf("BulkApply created a second path for existing ID: %s", derivedRenamedPath)
		}
	}
}

func TestBulkApplySerializesBackendInstances(t *testing.T) {
	dir := t.TempDir()
	configParent := t.TempDir()
	physicalParent := filepath.Join(configParent, "physical-parent")
	aliasParent := filepath.Join(configParent, "alias-parent")
	mustBulkApplyTest(t, "create physical parent", os.Mkdir(physicalParent, 0o755))
	mustBulkApplyTest(t, "create parent alias", os.Symlink(physicalParent, aliasParent))
	configDir := filepath.Join(physicalParent, "initially-absent-config")
	configAlias := filepath.Join(aliasParent, "initially-absent-config")
	initGitRepo(t, dir)
	first, err := gitlocal.NewWithConfigDir(dir, configAlias)
	mustBulkApplyTest(t, "new first backend through absent-path alias", err)
	second, err := gitlocal.NewWithConfigDir(dir, configDir)
	mustBulkApplyTest(t, "new second backend", err)

	entered := filepath.Join(t.TempDir(), "hook-entered")
	release := filepath.Join(t.TempDir(), "hook-release")
	hook := filepath.Join(dir, ".git", "hooks", "pre-commit")
	hookBody := fmt.Sprintf("#!/bin/sh\ntouch %q\nwhile [ ! -f %q ]; do sleep 0.01; done\n", entered, release)
	mustBulkApplyTest(t, "write blocking hook", os.WriteFile(hook, []byte(hookBody), 0o755))
	defer os.WriteFile(release, []byte("release"), 0o644)

	firstNote := makeNote("first concurrent")
	secondNote := makeNote("second concurrent")
	firstDone := make(chan error, 1)
	secondDone := make(chan error, 1)
	go func() { firstDone <- first.BulkApply([]*note.Note{firstNote}, nil) }()

	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, statErr := os.Stat(entered); statErr == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("first BulkApply did not reach the blocking commit hook")
		}
		time.Sleep(10 * time.Millisecond)
	}
	go func() { secondDone <- second.BulkApply([]*note.Note{secondNote}, nil) }()
	select {
	case earlyErr := <-secondDone:
		t.Fatalf("second BulkApply completed while first held transaction lock: %v", earlyErr)
	case <-time.After(200 * time.Millisecond):
	}
	mustBulkApplyTest(t, "release blocking hook", os.WriteFile(release, []byte("release"), 0o644))
	mustBulkApplyTest(t, "first BulkApply", <-firstDone)
	mustBulkApplyTest(t, "second BulkApply", <-secondDone)
	for _, n := range []*note.Note{firstNote, secondNote} {
		if out := bulkApplyGit(t, dir, "show", "HEAD:"+n.Filename()); !strings.Contains(out, n.Title) {
			t.Fatalf("committed serial outcome is missing %s", n.Title)
		}
	}
}

func TestBulkApplyRollsBackFailures(t *testing.T) {
	for _, mode := range []string{"write", "stage", "commit", "post-commit", "branch-switch", "symlink", "invalid-id", "collision", "nil-new", "nil-update"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			configDir := t.TempDir()
			initGitRepo(t, dir)
			b, err := gitlocal.NewWithConfigDir(dir, configDir)
			mustBulkApplyTest(t, "new backend", err)

			existing := makeNote("existing")
			existing.Body = "before"
			mustBulkApplyTest(t, "write existing note", b.Write(existing))
			mustBulkApplyTest(t, "preserve executable mode", os.Chmod(filepath.Join(dir, existing.Filename()), 0o4755))
			mustBulkApplyTest(t, "write unrelated file", os.WriteFile(filepath.Join(dir, "unrelated.txt"), []byte("staged before BulkApply"), 0o640))
			bulkApplyGit(t, dir, "add", "unrelated.txt")

			fresh := makeNote("fresh")
			fresh.Body = "new"
			updated := *existing
			updated.Body = "after"
			newNotes := []*note.Note{fresh}
			updates := []*note.Note{&updated}

			switch mode {
			case "write":
				blocked := makeNote("blocked")
				mustBulkApplyTest(t, "write blocked note", b.Write(blocked))
				blockedPath := filepath.Join(dir, blocked.Filename())
				mustBulkApplyTest(t, "remove blocked note", os.Remove(blockedPath))
				mustBulkApplyTest(t, "create blocked directory", os.Mkdir(blockedPath, 0o751))
				updates = append(updates, blocked)
			case "stage":
				realGit, lookupErr := exec.LookPath("git")
				mustBulkApplyTest(t, "find real git", lookupErr)
				shimDir := t.TempDir()
				shim := filepath.Join(shimDir, "git")
				script := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = add ] && [ \"$2\" = -N ]; then exit 72; fi\nexec %q \"$@\"\n", realGit)
				mustBulkApplyTest(t, "write git shim", os.WriteFile(shim, []byte(script), 0o755))
				t.Setenv("PATH", shimDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			case "commit":
				hook := filepath.Join(dir, ".git", "hooks", "pre-commit")
				mustBulkApplyTest(t, "write failing hook", os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o755))
			case "post-commit":
				realGit, lookupErr := exec.LookPath("git")
				mustBulkApplyTest(t, "find real git", lookupErr)
				shimDir := t.TempDir()
				shim := filepath.Join(shimDir, "git")
				script := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = commit ]; then\n  %q \"$@\"\n  status=$?\n  if [ $status -eq 0 ]; then exit 71; fi\n  exit $status\nfi\nexec %q \"$@\"\n", realGit, realGit)
				mustBulkApplyTest(t, "write git shim", os.WriteFile(shim, []byte(script), 0o755))
				t.Setenv("PATH", shimDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			case "branch-switch":
				realGit, lookupErr := exec.LookPath("git")
				mustBulkApplyTest(t, "find real git", lookupErr)
				shimDir := t.TempDir()
				shim := filepath.Join(shimDir, "git")
				script := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = commit ]; then\n  %q \"$@\"\n  status=$?\n  if [ $status -eq 0 ]; then %q switch -qc injected; exit 71; fi\n  exit $status\nfi\nexec %q \"$@\"\n", realGit, realGit, realGit)
				mustBulkApplyTest(t, "write branch-switch git shim", os.WriteFile(shim, []byte(script), 0o755))
				t.Setenv("PATH", shimDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			case "symlink":
				existingPath := filepath.Join(dir, existing.Filename())
				target := filepath.Join(dir, "symlink-target")
				mustBulkApplyTest(t, "write symlink target", os.WriteFile(target, []byte("target"), 0o600))
				mustBulkApplyTest(t, "remove existing for symlink", os.Remove(existingPath))
				mustBulkApplyTest(t, "create symlink", os.Symlink(target, existingPath))
			case "invalid-id":
				fresh.ID = "../outside-note"
				updates = nil
			case "collision":
				fresh.ID = existing.ID
				updates = nil
			case "nil-new":
				newNotes = append(newNotes, nil)
			case "nil-update":
				updates = append(updates, nil)
			}

			inputs := append(append([]*note.Note{}, newNotes...), updates...)
			nonNilInputs := inputs[:0]
			for _, input := range inputs {
				if input != nil {
					nonNilInputs = append(nonNilInputs, input)
				}
			}
			inputs = nonNilInputs
			before := captureBulkApplyRepoState(t, dir, configDir, inputs)
			var panicValue any
			func() {
				defer func() { panicValue = recover() }()
				err = b.BulkApply(newNotes, updates)
			}()
			if panicValue != nil {
				t.Fatalf("BulkApply panicked instead of rolling back: %v", panicValue)
			}
			if err == nil {
				t.Fatal("BulkApply error = nil, want injected failure")
			}
			if diff := diffBulkApplyRepoState(t, dir, configDir, inputs, before); diff != "" {
				t.Fatalf("BulkApply did not restore exact repository state: %s", diff)
			}
		})
	}
}
