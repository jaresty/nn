package cmd

import (
	"strings"
	"testing"
)

func TestTraceCmdBasic(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("trace", "../../../internal/trace", "--symbol", "BuildIndex")
	if err != nil {
		t.Fatalf("nn trace: %v", err)
	}
	if !strings.Contains(out, "BuildIndex") {
		t.Errorf("expected BuildIndex in output:\n%s", out)
	}
}

func TestTraceFooter(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("trace", "../../../internal/trace", "--symbol", "BuildIndex")
	if err != nil {
		t.Fatalf("nn trace: %v", err)
	}
	if !strings.Contains(out, "Continuing without resolving is a protocol violation.") {
		t.Errorf("expected resolution footer in human output:\n%s", out)
	}
	if !strings.Contains(out, "## Related notes") {
		t.Errorf("expected '## Related notes' section in human output:\n%s", out)
	}
}

func TestTraceFooterAbsentInJSON(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("trace", "../../../internal/trace", "--symbol", "BuildIndex", "--json")
	if err != nil {
		t.Fatalf("nn trace --json: %v", err)
	}
	if strings.Contains(out, "Continuing without resolving is a protocol violation.") {
		t.Errorf("footer must not appear in JSON output:\n%s", out)
	}
}

func TestTraceCmdJSON(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("trace", "../../../internal/trace", "--symbol", "BuildIndex", "--json")
	if err != nil {
		t.Fatalf("nn trace --json: %v", err)
	}
	if !strings.Contains(out, "nodes") {
		t.Errorf("expected 'nodes' in JSON output:\n%s", out)
	}
}
