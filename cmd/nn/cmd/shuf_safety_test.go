package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShufRecursiveSkipsBinaryContent(t *testing.T) {
	root := t.TempDir()
	textPath := filepath.Join(root, "safe.txt")
	binaryPath := filepath.Join(root, "opaque")
	writeTextErr := os.WriteFile(textPath, []byte("SAFE_TEXT\n"), 0o644)
	writeBinaryErr := os.WriteFile(binaryPath, []byte("BINARY_SENTINEL\x00TAIL\n"), 0o644)
	var out bytes.Buffer
	err := runShuf(nil, []string{root}, &out, nil, 10, "lines")
	got := out.String()
	if writeTextErr != nil || writeBinaryErr != nil || err != nil || got != "---\nSAFE_TEXT\n" {
		t.Fatalf("recursive shuf leaked binary content: writeTextErr=%v writeBinaryErr=%v err=%v output=%q", writeTextErr, writeBinaryErr, err, got)
	}
}

func TestShufTextOnlyOutputRemainsExact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "safe.txt")
	writeErr := os.WriteFile(path, []byte("SAFE_ONLY\n"), 0o644)
	var stdout, stderr bytes.Buffer
	cmd := newShufCmd(&rootState{})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{path, "--count", "1", "--unit", "lines"})
	err := cmd.Execute()
	if writeErr != nil || err != nil || stdout.String() != "---\nSAFE_ONLY\n" || stderr.String() != "" {
		t.Fatalf("text-only shuf output changed: writeErr=%v err=%v stdout=%q stderr=%q", writeErr, err, stdout.String(), stderr.String())
	}
}

func TestShufSkipsOversizedSamplePayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.txt")
	largeLine := strings.Repeat("A", 1000) + "\n"
	content := strings.Repeat(largeLine, 70) + "\nSAFE_SMALL\n"
	writeErr := os.WriteFile(path, []byte(content), 0o644)
	var out bytes.Buffer
	err := runShuf(nil, []string{path}, &out, nil, 10, "paragraphs")
	maxPayload := 0
	for _, rendered := range strings.Split(out.String(), "---\n") {
		payload := strings.TrimSuffix(rendered, "\n")
		if len(payload) > maxPayload {
			maxPayload = len(payload)
		}
	}
	if writeErr != nil || err != nil || maxPayload > 65536 || !strings.Contains(out.String(), "SAFE_SMALL") {
		t.Fatalf("oversized shuf payload emitted: writeErr=%v err=%v maxPayload=%d", writeErr, err, maxPayload)
	}
}

func TestShufReportsAggregateOversizedUnitCount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.txt")
	large := strings.Repeat(strings.Repeat("A", 1000)+"\n", 70)
	singleLongLine := strings.Repeat("B", 70000)
	content := large + "\n" + large + "\n" + singleLongLine + "\n\nSAFE_SMALL\n"
	_ = os.WriteFile(path, []byte(content), 0o644)
	var stdout, stderr bytes.Buffer
	cmd := newShufCmd(&rootState{})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{path, "--count", "10", "--unit", "paragraphs"})
	err := cmd.Execute()
	if err != nil || stdout.String() != "---\nSAFE_SMALL\n" || stderr.String() != "nn shuf: skipped 3 oversized units (>65536 bytes)\n" {
		t.Fatalf("oversized unit skip report incorrect: err=%v stdoutLen=%d stderr=%q", err, stdout.Len(), stderr.String())
	}
}

func TestShufDirectBinaryFileProducesNoStdout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opaque")
	_ = os.WriteFile(path, []byte("DIRECT_BINARY\x00TAIL\n"), 0o644)
	var stdout, stderr bytes.Buffer
	cmd := newShufCmd(&rootState{})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{path, "--count", "1", "--unit", "lines"})
	err := cmd.Execute()
	if err != nil || stdout.Len() != 0 {
		t.Fatalf("direct binary file reached stdout: err=%v stdout=%q", err, stdout.String())
	}
}

func TestShufDirectBinaryFileReportsOneSkip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opaque")
	_ = os.WriteFile(path, []byte("DIRECT_BINARY\x00TAIL\n"), 0o644)
	var stdout, stderr bytes.Buffer
	cmd := newShufCmd(&rootState{})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{path, "--count", "1", "--unit", "lines"})
	err := cmd.Execute()
	if err != nil || stderr.String() != "nn shuf: skipped 1 binary file\n" {
		t.Fatalf("direct binary skip report incorrect: err=%v stderr=%q", err, stderr.String())
	}
}

func TestShufHelpDocumentsBinaryFiltering(t *testing.T) {
	cmd := newShufCmd(&rootState{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	if err != nil || !strings.Contains(strings.ToLower(out.String()), "binary files") {
		t.Fatalf("shuf help missing binary filtering: err=%v output=%q", err, out.String())
	}
}

func TestShufHelpDocumentsUnitByteLimit(t *testing.T) {
	cmd := newShufCmd(&rootState{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	if err != nil || !strings.Contains(out.String(), "65536") {
		t.Fatalf("shuf help missing unit byte limit: err=%v output=%q", err, out.String())
	}
}

func TestNNGuideDocumentsShufBinaryFiltering(t *testing.T) {
	guide, readErr := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	section := string(guide)
	if start := strings.Index(section, "## nn shuf"); start >= 0 {
		section = section[start:]
		if end := strings.Index(section[1:], "\n## "); end >= 0 {
			section = section[:end+1]
		}
	}
	if readErr != nil || !strings.Contains(section, "binary files") {
		t.Fatalf("nn-guide shuf section missing binary filtering: readErr=%v", readErr)
	}
}

func TestNNGuideDocumentsShufUnitLimit(t *testing.T) {
	guide, readErr := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	section := string(guide)
	if start := strings.Index(section, "## nn shuf"); start >= 0 {
		section = section[start:]
		if end := strings.Index(section[1:], "\n## "); end >= 0 {
			section = section[:end+1]
		}
	}
	if readErr != nil || !strings.Contains(section, "65536") {
		t.Fatalf("nn-guide shuf section missing unit limit: readErr=%v", readErr)
	}
}

func TestShufSkipsBinaryMarkerBeyondMIMESniffWindow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "late-binary")
	payload := append([]byte(strings.Repeat("A", 600)), []byte("LATE_BINARY\x00TAIL\n")...)
	_ = os.WriteFile(path, payload, 0o644)
	var stdout, stderr bytes.Buffer
	cmd := newShufCmd(&rootState{})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{path, "--count", "1", "--unit", "lines"})
	err := cmd.Execute()
	if err != nil || stdout.Len() != 0 || stderr.String() != "nn shuf: skipped 1 binary file\n" {
		t.Fatalf("late binary marker reached shuf output: err=%v stdoutLen=%d stderr=%q", err, stdout.Len(), stderr.String())
	}
}

func TestShufAllOversizedUnitsProduceNoStdout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "only-large.txt")
	_ = os.WriteFile(path, []byte(strings.Repeat("X", 70000)+"\n"), 0o644)
	var stdout, stderr bytes.Buffer
	cmd := newShufCmd(&rootState{})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{path, "--count", "1", "--unit", "lines"})
	err := cmd.Execute()
	if err != nil || stdout.Len() != 0 {
		t.Fatalf("all-oversized shuf emitted stdout: err=%v stdoutLen=%d", err, stdout.Len())
	}
}

func TestShufOneOversizedUnitUsesSingularDiagnostic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "only-large.txt")
	_ = os.WriteFile(path, []byte(strings.Repeat("X", 70000)+"\n"), 0o644)
	var stdout, stderr bytes.Buffer
	cmd := newShufCmd(&rootState{})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{path, "--count", "1", "--unit", "lines"})
	err := cmd.Execute()
	if err != nil || stderr.String() != "nn shuf: skipped 1 oversized unit (>65536 bytes)\n" {
		t.Fatalf("singular oversized diagnostic incorrect: err=%v stderr=%q", err, stderr.String())
	}
}

func TestShufSkipsLateInvalidUTF8(t *testing.T) {
	path := filepath.Join(t.TempDir(), "late-invalid")
	payload := append([]byte(strings.Repeat("A", 600)), 0xff, 0xfe)
	_ = os.WriteFile(path, payload, 0o644)
	var stdout, stderr bytes.Buffer
	cmd := newShufCmd(&rootState{})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{path, "--count", "1", "--unit", "lines"})
	err := cmd.Execute()
	if err != nil || stdout.Len() != 0 || stderr.String() != "nn shuf: skipped 1 binary file\n" {
		t.Fatalf("late invalid UTF-8 reached shuf output: err=%v stdoutLen=%d stderr=%q", err, stdout.Len(), stderr.String())
	}
}

func TestShufSkipsLateBinaryControlBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "late-control")
	payload := append([]byte(strings.Repeat("A", 600)), 0x01, 0x02)
	_ = os.WriteFile(path, payload, 0o644)
	var stdout, stderr bytes.Buffer
	cmd := newShufCmd(&rootState{})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{path, "--count", "1", "--unit", "lines"})
	err := cmd.Execute()
	if err != nil || stdout.Len() != 0 || stderr.String() != "nn shuf: skipped 1 binary file\n" {
		t.Fatalf("late control bytes reached shuf output: err=%v stdoutLen=%d stderr=%q", err, stdout.Len(), stderr.String())
	}
}

func TestShufSkipsWhitespacePrefixedPDF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prefixed-document")
	payload := []byte("\n\t%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\n%%EOF\n")
	_ = os.WriteFile(path, payload, 0o644)
	var stdout, stderr bytes.Buffer
	cmd := newShufCmd(&rootState{})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{path, "--count", "10", "--unit", "lines"})
	err := cmd.Execute()
	if err != nil || stdout.Len() != 0 || stderr.String() != "nn shuf: skipped 1 binary file\n" {
		t.Fatalf("whitespace-prefixed PDF reached shuf output: err=%v stdoutLen=%d stderr=%q", err, stdout.Len(), stderr.String())
	}
}

func TestShufHelpDocumentsStdinMIMEException(t *testing.T) {
	cmd := newShufCmd(&rootState{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	if err != nil || !strings.Contains(out.String(), "stdin is not MIME-filtered") {
		t.Fatalf("shuf help missing stdin MIME exception: err=%v output=%q", err, out.String())
	}
}

func TestNNGuideDocumentsShufStdinException(t *testing.T) {
	guide, readErr := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	section := string(guide)
	if start := strings.Index(section, "## nn shuf"); start >= 0 {
		section = section[start:]
		if end := strings.Index(section[1:], "\n## "); end >= 0 {
			section = section[:end+1]
		}
	}
	if readErr != nil || !strings.Contains(section, "stdin is not MIME-filtered") {
		t.Fatalf("nn-guide shuf section missing stdin exception: readErr=%v", readErr)
	}
}

func TestNNGuideDocumentsRecursiveShufDirectoryExample(t *testing.T) {
	guide, readErr := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	text := string(guide)
	if readErr != nil || !strings.Contains(text, "nn shuf . --count 10 --unit symbols") || !strings.Contains(text, "recursively") || strings.Contains(text, "won't work — pass file paths") {
		t.Fatalf("nn-guide shuf directory example is stale: readErr=%v", readErr)
	}
}

func TestShufReportsAggregateBinarySkipCount(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "first.bin"), []byte("FIRST\x00"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "second"), []byte("SECOND\x00"), 0o644)
	var stdout, stderr bytes.Buffer
	cmd := newShufCmd(&rootState{})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{root, "--count", "10", "--unit", "lines"})
	err := cmd.Execute()
	if err != nil || stderr.String() != "nn shuf: skipped 2 binary files\n" {
		t.Fatalf("binary skip report incorrect: err=%v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
}
