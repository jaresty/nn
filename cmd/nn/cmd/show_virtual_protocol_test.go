package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: nn show --global always includes virtual protocols even when notebook is empty.
func TestShowGlobalVirtualAlwaysPresent(t *testing.T) {
	_, execute := setupNotebook(t)
	// Empty notebook — no notes written.

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "virtual-nn-capture-discipline") {
		t.Errorf("expected virtual-nn-capture-discipline id in output:\n%s", out)
	}
	// Compact format: body is not in --global; fetch via nn show <id>
	out2, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out2, "Gate: Search rationale:") {
		t.Errorf("expected virtual protocol body text in nn show output:\n%s", out2)
	}
}

// Assertion: nn show --global virtual protocols appear alongside real notebook protocols.
func TestShowGlobalVirtualAppearsWithReal(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	real := newTestNoteForCLI(note.GenerateID(), "Real Protocol", note.TypeProtocol)
	writeNoteFile(t, nbDir, real)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "Real Protocol") {
		t.Errorf("expected real protocol in output:\n%s", out)
	}
	if !strings.Contains(out, "virtual-nn-capture-discipline") {
		t.Errorf("expected virtual protocol in output alongside real:\n%s", out)
	}
}

// Assertion: virtual-nn-error-handling appears in nn show --global output.
func TestShowVirtualErrorHandlingGlobal(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "virtual-nn-error-handling") {
		t.Errorf("expected virtual-nn-error-handling id in output:\n%s", out)
	}
}

// Assertion: nn show virtual-nn-error-handling returns body text.
func TestShowVirtualErrorHandlingBody(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "virtual-nn-error-handling")
	if err != nil {
		t.Fatalf("nn show virtual-nn-error-handling: %v", err)
	}
	if !strings.Contains(out, "Skip condition B") {
		t.Errorf("expected virtual error-handling body text in output:\n%s", out)
	}
}

// Assertion: error-handling skip clause is present and requires verbatim citation.
func TestShowVirtualErrorHandlingSkipClause(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "virtual-nn-error-handling")
	if err != nil {
		t.Fatalf("nn show virtual-nn-error-handling: %v", err)
	}
	if !strings.Contains(out, "Expected FAIL:") {
		t.Errorf("expected skip condition referencing 'Expected FAIL:' in output:\n%s", out)
	}
	if !strings.Contains(out, "verbatim") {
		t.Errorf("expected skip condition to require verbatim citation:\n%s", out)
	}
}

// Assertion: virtual-nn-cli-reference appears in nn show --global output.
func TestShowVirtualCLIReferenceGlobal(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "virtual-nn-cli-reference") {
		t.Errorf("expected virtual-nn-cli-reference id in output:\n%s", out)
	}
}

// Assertion: nn show virtual-nn-cli-reference returns body covering valid types and statuses.
func TestShowVirtualCLIReferenceBody(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("nn show virtual-nn-cli-reference: %v", err)
	}
	if !strings.Contains(out, "concept|argument|model|hypothesis|observation|question|protocol") {
		t.Errorf("expected valid type values in body:\n%s", out)
	}
	if !strings.Contains(out, "draft|reviewed|permanent") {
		t.Errorf("expected valid status values in body:\n%s", out)
	}
	if !strings.Contains(out, "refines|contradicts|source-of|extends|supports|grounded-by|questions|governs|requires") {
		t.Errorf("expected valid link type values in body:\n%s", out)
	}
}

// Assertion: virtual-nn-cli-reference body covers graph exploration commands and points to /nn-guide.
func TestShowVirtualCLIReferenceGraphCommands(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("nn show virtual-nn-cli-reference: %v", err)
	}
	if !strings.Contains(out, "nn graph") {
		t.Errorf("expected nn graph command in body:\n%s", out)
	}
	if !strings.Contains(out, "nn path") {
		t.Errorf("expected nn path command in body:\n%s", out)
	}
	if !strings.Contains(out, "nn clusters") {
		t.Errorf("expected nn clusters command in body:\n%s", out)
	}
	if !strings.Contains(out, "--similar") {
		t.Errorf("expected nn list --similar in body:\n%s", out)
	}
	if !strings.Contains(out, "--depth") {
		t.Errorf("expected nn show --depth in body:\n%s", out)
	}
	if !strings.Contains(out, "/nn-guide") {
		t.Errorf("expected /nn-guide pointer in body:\n%s", out)
	}
	if !strings.Contains(out, "nn graph show") {
		t.Errorf("expected nn graph show subcommand in body:\n%s", out)
	}
	if !strings.Contains(out, "--focus") {
		t.Errorf("expected --focus flag in body:\n%s", out)
	}
}

// Assertion: virtual-nn-cli-reference surfaces nn trace as symbol-discovery alternative to grep.
func TestShowVirtualCLIReferenceTrace(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("nn show virtual-nn-cli-reference: %v", err)
	}
	if !strings.Contains(out, "nn trace") {
		t.Errorf("expected nn trace in virtual-nn-cli-reference body:\n%s", out)
	}
	if !strings.Contains(out, "--symbol") {
		t.Errorf("expected --symbol flag documented in virtual-nn-cli-reference body:\n%s", out)
	}
}

// Assertion: virtual-nn-capture-discipline allow-list includes nn trace.
func TestCaptureDisciplineAllowListIncludesTrace(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "nn trace") {
		t.Errorf("expected nn trace in capture-discipline allow-list:\n%s", out)
	}
}

// Assertion: empty/truncated exit requires skip-search: prefix with a quoted substring from the tool result.
func TestCaptureDisciplineSkipClauseRequiresVerbatimExcerpt(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "skip-search:") {
		t.Errorf("expected 'skip-search:' exit for empty/truncated output:\n%s", out)
	}
	if !strings.Contains(out, "appears verbatim in the tool result") {
		t.Errorf("expected requirement that quoted substring appears verbatim in tool result:\n%s", out)
	}
}

// Assertion: virtual-nn-capture-discipline gate instruction does not tell agents to use --show-first.
func TestShowCaptureDisciplineNoShowFirst(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if strings.Contains(out, "requires a preceding `nn list --search") && strings.Contains(out, "--show-first") {
		// Check that the gate instruction sentence itself doesn't contain --show-first.
		for line := range strings.SplitSeq(out, "\n") {
			if strings.Contains(line, "requires a preceding") && strings.Contains(line, "--show-first") {
				t.Errorf("gate instruction must not tell agents to use --show-first: %q", line)
			}
		}
	}
}

// Assertion: allow-list clause names specific write-indicating strings rather than broad "Bash tool call".
func TestShowCaptureDisciplineAllowListWriteStrings(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "Write, Edit") {
		t.Errorf("allow-list clause must reference Write and Edit tool calls:\n%s", out)
	}
}

// Assertion: virtual-nn-cli-reference body prohibits piping nn show output to truncating commands.
func TestShowVirtualCLIReferenceNoPipeTruncate(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("nn show virtual-nn-cli-reference: %v", err)
	}
	if !strings.Contains(out, "must not be piped to") {
		t.Errorf("expected pipe prohibition in cli-reference body:\n%s", out)
	}
	for _, cmd := range []string{"head", "tail", "less", "more"} {
		if !strings.Contains(out, "`"+cmd+"`") {
			t.Errorf("expected %q named in pipe prohibition:\n%s", cmd, out)
		}
	}
	if !strings.Contains(out, "complete note body is required") {
		t.Errorf("expected rationale clause in pipe prohibition:\n%s", out)
	}
}

// Assertion: virtual-nn-capture-discipline prohibits piping nn list --search and directs use of --limit N.
func TestShowCaptureDisciplineNoPipeDirective(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "no additional flags, arguments, or shell operators") {
		t.Errorf("expected pipe prohibition in capture-discipline body:\n%s", out)
	}
}

// Assertion: virtual-nn-capture-discipline requires nn show on the best-fit result via Selected because: judgment.
func TestShowCaptureDisciplineRequiresShowOnWordMatch(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "Selected because:") {
		t.Errorf("expected 'Selected because:' judgment requirement before nn show:\n%s", out)
	}
	if !strings.Contains(out, "nn show") {
		t.Errorf("expected nn show command in capture-discipline body:\n%s", out)
	}
}

// Assertion: virtual-nn-cli-reference body documents nn promote with conflict detection behavior.
func TestShowVirtualCLIReferencePromote(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("nn show virtual-nn-cli-reference: %v", err)
	}
	if !strings.Contains(out, "nn promote") {
		t.Errorf("expected nn promote command in cli-reference body:\n%s", out)
	}
	if !strings.Contains(out, "--to") {
		t.Errorf("expected --to flag for nn promote in cli-reference body:\n%s", out)
	}
	if !strings.Contains(out, "conflict") {
		t.Errorf("expected conflict detection note for nn promote in cli-reference body:\n%s", out)
	}
}

// Assertion: virtual-nn-cli-reference body documents nn ast command.
func TestShowVirtualCLIReferenceAst(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("nn show virtual-nn-cli-reference: %v", err)
	}
	if !strings.Contains(out, "nn ast") {
		t.Errorf("expected nn ast in virtual-nn-cli-reference body:\n%s", out)
	}
	if !strings.Contains(out, "--refs") {
		t.Errorf("expected --refs flag documented in virtual-nn-cli-reference body:\n%s", out)
	}
}

// Assertion: virtual-nn-capture-discipline allow-list includes nn ast.
func TestCaptureDisciplineAllowListIncludesAst(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "nn ast") {
		t.Errorf("expected nn ast in capture-discipline allow-list:\n%s", out)
	}
}

// Assertion: virtual-nn-cli-reference body documents that --content is literal and recommends $'...' or stdin.
func TestShowVirtualCLIReferenceContentLiteral(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("nn show virtual-nn-cli-reference: %v", err)
	}
	if !strings.Contains(out, "--content` is literal") {
		t.Errorf("expected --content literal note in cli-reference body:\n%s", out)
	}
	if !strings.Contains(out, `$'...'`) {
		t.Errorf("expected $'...' ANSI-C quoting recommendation in cli-reference body:\n%s", out)
	}
}
