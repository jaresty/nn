package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGrepContextReportDefaultOutputUnchanged(t *testing.T) {
	_, execute := setupNotebook(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	writeErr := os.WriteFile(path, []byte("alpha\nTARGET\nomega\n"), 0o644)
	out, err := execute("grep", "TARGET", path, "--context", "1")
	want := "==> " + path + " <==\n1:alpha\n2:TARGET\n3:omega\n"
	if writeErr != nil || err != nil || out != want {
		t.Fatalf("default grep output changed: writeErr=%v err=%v\ngot=%q\nwant=%q", writeErr, err, out, want)
	}
}

func TestGrepContextReportCountsRetainedBlocks(t *testing.T) {
	_, execute := setupNotebook(t)
	path := filepath.Join(t.TempDir(), "overlap.txt")
	_ = os.WriteFile(path, []byte("one\nMATCH A\nshared\nMATCH B\nfive\n"), 0o644)

	out, err := execute("grep", "MATCH", path, "--context", "1", "--context-report")
	if err != nil || !strings.Contains(out, "context blocks: 2\n") {
		t.Fatalf("context block report incorrect: err=%v output=%q", err, out)
	}
}

func TestGrepContextReportCountsGrossLines(t *testing.T) {
	_, execute := setupNotebook(t)
	path := filepath.Join(t.TempDir(), "overlap.txt")
	_ = os.WriteFile(path, []byte("one\nMATCH A\nshared\nMATCH B\nfive\n"), 0o644)

	out, err := execute("grep", "MATCH", path, "--context", "1", "--context-report")
	if err != nil || !strings.Contains(out, "gross context lines: 6\n") {
		t.Fatalf("gross context line report incorrect: err=%v output=%q", err, out)
	}
}

func TestGrepContextReportCountsUniqueLinesByPath(t *testing.T) {
	_, execute := setupNotebook(t)
	path := filepath.Join(t.TempDir(), "overlap.txt")
	_ = os.WriteFile(path, []byte("one\nMATCH A\nshared\nMATCH B\nfive\n"), 0o644)

	out, err := execute("grep", "MATCH", path, "--context", "1", "--context-report")
	if err != nil || !strings.Contains(out, "unique context lines: 5\n") {
		t.Fatalf("unique context line report incorrect: err=%v output=%q", err, out)
	}
}

func TestGrepContextReportDerivesOverlapLines(t *testing.T) {
	_, execute := setupNotebook(t)
	path := filepath.Join(t.TempDir(), "overlap.txt")
	_ = os.WriteFile(path, []byte("one\nMATCH A\nshared\nMATCH B\nfive\n"), 0o644)

	out, err := execute("grep", "MATCH", path, "--context", "1", "--context-report")
	if err != nil || !strings.Contains(out, "overlap context lines: 1\n") {
		t.Fatalf("overlap context line report incorrect: err=%v output=%q", err, out)
	}
}

func TestGrepContextReportIsDeterministic(t *testing.T) {
	_, execute := setupNotebook(t)
	path := filepath.Join(t.TempDir(), "overlap.txt")
	_ = os.WriteFile(path, []byte("one\nMATCH A\nshared\nMATCH B\nfive\n"), 0o644)

	first, firstErr := execute("grep", "MATCH", path, "--context", "1", "--context-report")
	second, secondErr := execute("grep", "MATCH", path, "--context", "1", "--context-report")
	if firstErr != nil || secondErr != nil || first != second {
		t.Fatalf("context report is nondeterministic: firstErr=%v secondErr=%v first=%q second=%q", firstErr, secondErr, first, second)
	}
}

func TestGrepContextReportReportsZeroBlocks(t *testing.T) {
	_, execute := setupNotebook(t)
	path := filepath.Join(t.TempDir(), "none.txt")
	_ = os.WriteFile(path, []byte("no matches here\n"), 0o644)

	out, err := execute("grep", "ABSENT", path, "--context-report")
	if err != nil || !strings.Contains(out, "context blocks: 0\n") {
		t.Fatalf("zero context block report incorrect: err=%v output=%q", err, out)
	}
}

func TestGrepContextReportReportsZeroGrossLines(t *testing.T) {
	_, execute := setupNotebook(t)
	path := filepath.Join(t.TempDir(), "none.txt")
	_ = os.WriteFile(path, []byte("no matches here\n"), 0o644)

	out, err := execute("grep", "ABSENT", path, "--context-report")
	if err != nil || !strings.Contains(out, "gross context lines: 0\n") {
		t.Fatalf("zero gross context line report incorrect: err=%v output=%q", err, out)
	}
}

func TestGrepContextReportReportsZeroUniqueLines(t *testing.T) {
	_, execute := setupNotebook(t)
	path := filepath.Join(t.TempDir(), "none.txt")
	_ = os.WriteFile(path, []byte("no matches here\n"), 0o644)

	out, err := execute("grep", "ABSENT", path, "--context-report")
	if err != nil || !strings.Contains(out, "unique context lines: 0\n") {
		t.Fatalf("zero unique context line report incorrect: err=%v output=%q", err, out)
	}
}

func TestGrepContextReportReportsZeroOverlapLines(t *testing.T) {
	_, execute := setupNotebook(t)
	path := filepath.Join(t.TempDir(), "none.txt")
	_ = os.WriteFile(path, []byte("no matches here\n"), 0o644)

	out, err := execute("grep", "ABSENT", path, "--context-report")
	if err != nil || !strings.Contains(out, "overlap context lines: 0\n") {
		t.Fatalf("zero overlap context line report incorrect: err=%v output=%q", err, out)
	}
}

func TestGrepContextReportUsesRetainedMatchPopulation(t *testing.T) {
	_, execute := setupNotebook(t)
	path := filepath.Join(t.TempDir(), "truncated.txt")
	_ = os.WriteFile(path, []byte("MATCH A\nMATCH B\nMATCH C\n"), 0o644)

	out, err := execute("grep", "MATCH", path, "--context", "0", "--max-matches", "2", "--context-report")
	if err != nil || !strings.Contains(out, "truncated: 1 more matches") || !strings.Contains(out, "context blocks: 2\n") {
		t.Fatalf("retained context block population incorrect: err=%v output=%q", err, out)
	}
}

func TestGrepContextReportUniqueIdentityIncludesPath(t *testing.T) {
	_, execute := setupNotebook(t)
	dir := t.TempDir()
	first := filepath.Join(dir, "first.txt")
	second := filepath.Join(dir, "second.txt")
	_ = os.WriteFile(first, []byte("MATCH\n"), 0o644)
	_ = os.WriteFile(second, []byte("MATCH\n"), 0o644)

	out, err := execute("grep", "MATCH", first, second, "--context", "0", "--context-report")
	if err != nil || !strings.Contains(out, "unique context lines: 2\n") || !strings.Contains(out, "overlap context lines: 0\n") {
		t.Fatalf("source line identity omitted path: err=%v output=%q", err, out)
	}
}

func TestNNGuideDocumentsGrepContextReport(t *testing.T) {
	guide, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil || !strings.Contains(string(guide), "## nn grep") || !strings.Contains(string(guide), "--context-report") {
		t.Fatalf("nn-guide missing grep context report: err=%v", err)
	}
}

func TestGrepContextReportFlagIsDocumented(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("grep", "--help")
	if err != nil || !strings.Contains(out, "--context-report") || !strings.Contains(out, "source context block and overlap metrics") {
		t.Fatalf("grep context-report help missing: err=%v output=%q", err, out)
	}
}
